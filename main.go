package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors" // Importar el middleware CORS
	"github.com/joho/godotenv"
	"gonum.org/v1/gonum/mat"
)

// rotateMatrix rota una matriz 90° en sentido horario.
// Recibe una matriz bidimensional y devuelve una nueva matriz que es la rotación
// de la original. Se realiza intercambiando filas por columnas, invirtiendo el orden
// de las filas para lograr el efecto de rotación en el sentido de las agujas del reloj.
func rotateMatrix(matrix [][]float64) [][]float64 {
	rows := len(matrix)
	cols := len(matrix[0])
	rotated := make([][]float64, cols)

	for i := 0; i < cols; i++ {
		rotated[i] = make([]float64, rows)
		for j := 0; j < rows; j++ {
			rotated[i][j] = matrix[rows-j-1][i]
		}
	}

	return rotated
}

// main configura y arranca el servidor, configura el middleware CORS y define
// el endpoint POST /qr que procesa la factorización QR de una matriz recibida.
func main() {
	app := fiber.New()

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	// Configurar CORS para permitir solicitudes desde el front-end (localhost:5173)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		// os.Getenv("CORS_FRONT"), // Permitir solicitudes solo desde esta URL
		AllowMethods: "GET,POST,PUT,DELETE", // Métodos permitidos
		AllowHeaders: "Content-Type",        // Encabezados permitidos
	}))

	// Endpoint para realizar la factorización QR y enviar datos a Node.js
	app.Post("/qr", func(c *fiber.Ctx) error {
		// Leer la matriz enviada en JSON
		var input [][]float64
		if err := json.Unmarshal(c.Body(), &input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Entrada no válida, se esperaba una matriz bidimensional"})
		}

		// Verificar que la matriz no esté vacía
		if len(input) == 0 || len(input[0]) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "La matriz no puede estar vacía"})
		}

		// Rotar la matriz
		rotatedMatrix := rotateMatrix(input)

		// Convertir la matriz rotada en una densa de Gonum
		rows := len(rotatedMatrix)
		cols := len(rotatedMatrix[0])
		data := make([]float64, 0, rows*cols)
		for _, row := range rotatedMatrix {
			data = append(data, row...)
		}
		matrix := mat.NewDense(rows, cols, data)

		// Realizar la factorización QR
		var qr mat.QR
		qr.Factorize(matrix)

		// Obtener las matrices Q y R
		var q mat.Dense
		var r mat.Dense
		qr.QTo(&q)
		qr.RTo(&r)

		// Normalizar R para que tenga valores positivos en la diagonal
		qRows, qCols := q.Dims()
		rRows, rCols := r.Dims()
		for i := 0; i < int(math.Min(float64(qCols), float64(rRows))); i++ {
			if r.At(i, i) < 0 {
				for j := 0; j < qRows; j++ {
					q.Set(j, i, -q.At(j, i))
				}
				for j := 0; j < rCols; j++ {
					r.Set(i, j, -r.At(i, j))
				}
			}
		}

		// Convertir las matrices Q y R en arreglos bidimensionales
		qData := make([][]float64, qRows)
		rData := make([][]float64, rRows)
		for i := 0; i < qRows; i++ {
			qData[i] = make([]float64, qCols)
			for j := 0; j < qCols; j++ {
				qData[i][j] = q.At(i, j)
			}
		}
		for i := 0; i < rRows; i++ {
			rData[i] = make([]float64, rCols)
			for j := 0; j < rCols; j++ {
				rData[i][j] = r.At(i, j)
			}
		}

		// Crear el payload para la API Node.js, incluyendo las matrices Q, R y la matriz rotada
		payload := map[string]interface{}{
			"Q":       qData,
			"R":       rData,
			"rotated": rotatedMatrix, // Incluir la matriz rotada en el payload
		}

		// Convertir el payload a JSON
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Fallo al procesar datos"})
		}

		// Enviar el payload a la API de Node.js
		resp, err := http.Post(os.Getenv("API_EXPRESS")+"/analyze", "application/json", bytes.NewBuffer(payloadJSON))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Fallo al comunicarse con el servidor Express"})
		}
		defer resp.Body.Close()

		// Leer la respuesta de la API de Node.js
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Fallo el formateo del JSON del servidor Express"})
		}

		// Devolver el JSON final al cliente
		return c.JSON(result)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Valor predeterminado si no se encuentra la variable PORT
	}
	log.Fatal(app.Listen(":" + port))
}
