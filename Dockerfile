# Utilizar la imagen base de Go (Alpine)
FROM golang:1.23-alpine

# Instalar dependencias necesarias y herramientas para la compilación
RUN apk add --no-cache \
    bash \
    git \
    make \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Crear y establecer el directorio de trabajo
WORKDIR /app

# Copiar los archivos go.mod y go.sum al contenedor
COPY go.mod go.sum ./

# Descargar las dependencias
RUN go mod tidy

# Copiar el código fuente de la aplicación al contenedor
COPY . .

# Exponer el puerto 8080 para que sea accesible desde fuera del contenedor
EXPOSE 8080

# Cargar las variables de entorno desde el archivo .env (si es necesario)
COPY .env ./

# Ejecutar la aplicación Go
CMD ["go", "run", "main.go"]
