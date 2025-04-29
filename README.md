# fullcycle-observability-otel

Este projeto contém dois serviços:

- Service A: Recebe o input do usuário (CEP via POST) e chama o Service B.
- Service B: Orquestra a consulta do nome da cidade via ViaCEP e do clima via WeatherAPI.
- Zipkin: Ferramenta para visualizar os traces distribuídos (Service A ↔ Service B).

## 🛠 Como Rodar o Projeto

Certifique-se que você tenha instalado:

- Docker
- Docker Compose
- Make (ou execute os comandos manualmente)


## 📄 Comandos principais

1. Build das imagens Docker:

```
make build
```

2. Subir os containers
```
make up
```
Obs: o make up fará docker-compose up automaticamente.

## 🚀 Serviços Disponíveis

```
Serviço	URL de Acesso	            Descrição
- Service A	http://localhost:8081	Recebe POST com CEP
- Service B	http://localhost:8080	Endpoint interno chamado pelo A
- Zipkin UI	http://localhost:9411	Visualizar os traces
```

## 📬 Como fazer uma Requisição

Endpoint do Service A
- URL: http://localhost:8081/request-weather
- Método: POST
- Body JSON:

```
curl -X POST http://localhost:8081/request-weather \
  -H "Content-Type: application/json" \
  -d '{"cep":"29216090"}'
```


## 🔍 Acessando o Zipkin UI

Após executar o make up, acesse:

- Zipkin UI: http://localhost:9411
Dentro do painel:

- Clique em Run Query para listar todos os traces.
- Você verá o fluxo:
    - Service A chamando o Service B.
    - Service B realizando a busca no ViaCEP e WeatherAPI.

## 📦 Estrutura do Projeto
```
├── docker-compose.yml
├── Makefile
├── service-a/
│   ├── main.go
│   └── Dockerfile
├── service-b/
│   ├── main.go
│   └── Dockerfile
└── README.md
```

## 📜 Makefile

Exemplo de Makefile que você pode usar:

```
build:
	docker-compose build

up:
	docker-compose up
```

## 📢 Observações importantes

- As variáveis de ambiente necessárias (SERVICE_B_URL, ZIPKIN_ENDPOINT, WEATHER_API_KEY) já estão configuradas no docker-compose.yml.
- O OpenTelemetry foi implementado para instrumentar tanto Service A quanto Service B.
- O trace é propagado automaticamente entre as chamadas HTTP usando otel + propagation.