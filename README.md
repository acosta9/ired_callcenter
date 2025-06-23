# README #

Microservicio backend en Go que actúa como el sistema nervioso central para las operaciones de un call center. Este servicio no es solo una API REST, sino un orquestador que integra en tiempo real sistemas heterogéneos y automatiza procesos críticos del negocio.

El núcleo del proyecto gestiona la comunicación en tiempo real con una central telefónica Asterisk a través de su interfaz AMI, procesando eventos de llamadas (entrantes, atendidas, colgadas) de forma asíncrona mediante goroutines y canales. Esto permite, por ejemplo, registrar cada llamada en una base de datos MySQL y exponer el estado de las extensiones a sistemas de monitoreo como Grafana.

Paralelamente, el servicio se conecta a una API externa de Chatwoot y a su base de datos PostgreSQL para ejecutar tareas de mantenimiento automatizadas, como resolver o reabrir conversaciones de chat basándose en reglas de negocio complejas, todo orquestado a través de un sistema de cronjobs integrado.

El diseño sigue una arquitectura limpia y desacoplada (controladores, repositorios, modelos), es robusto para producción con logging rotativo, gestión de pánicos y configuración por variables de entorno, y está completamente documentado vía Swagger para facilitar su integración.

## 🛠 Tecnologías utilizadas

* Arquitectura de Microservicios e Integración: No es solo una API. Es un servicio que conecta y media entre un PBX (Asterisk), dos bases de datos distintas (MySQL, PostgreSQL) y una API REST externa (Chatwoot).
* Concurrencia y Asincronía: El manejo de eventos de Asterisk AMI (repo/cronServiceRepo.go y repo/grafanaRepo.go) es un ejemplo perfecto de concurrencia. Se usa un bucle infinito con un select sobre canales, una práctica idiomática en Go para manejar flujos de datos asíncronos de manera eficiente y sin bloqueos.

Robustez y Preparación para Producción:
* Logging: Se utiliza lumberjack para la rotación de logs, una necesidad en cualquier aplicación de larga duración.
* Gestión de Errores y Pánicos: gin.RecoveryWithWriter asegura que el servicio no se caiga ante un pánico inesperado, sino que lo registre. Contexto (context.Context): Se utiliza en todo el proyecto context.WithTimeout en las operaciones de base de datos y en las interacciones con AMI. Esto es crucial para evitar que el sistema se quede "colgado" por una dependencia lenta y de tal manera que sea un servicio resiliente a cambios externos.

Gestión de Dependencias y Configuración:
* Pool de Conexiones: En app/dbApp.go, se configura explícitamente los pools de conexiones para MySQL y PostgreSQL. Esto es fundamental para el rendimiento y la escalabilidad.
* Variables de Entorno (.env): Siguiendo las mejores prácticas de "12-Factor App" al separar la configuración del código, lo que facilita el despliegue en diferentes entornos (desarrollo, producción).

Calidad y Mantenibilidad del Código:
* API Documentada: El uso de swagger para generar documentación a partir de comentarios demuestra un compromiso con la creación de APIs fáciles de consumir y mantener.
* Estructura del Proyecto: La separación en paquetes (controllers, repo, models, app) es clara y facilita la navegación y la adición de nuevas funcionalidades.

### Estatus en tiempo real del callcenter a travès de grafana, esta info es posible mostrarla en grafana debido al servicio construido que permite extraer en tiempo real la informacion de las extensiones, colas de llamadas, y operadores  
![Screenshot From 2025-06-23 10-50-01](https://github.com/user-attachments/assets/b4c52653-7ad4-4f83-839a-af5d876bc9cb)

### Gestion Operativa del callcenter, despliegue de indicadores del callcenter.
![screencapture-grafana-bessersolutions-d-bel2ruzz561vkf-central-telefonica-indicadores-de-gestion-2025-06-23-10_51_32](https://github.com/user-attachments/assets/6e1a1fb6-c648-4250-bfb0-9af914be7652)


### this project contains the next tasks ###
* project to handle all call center related tasks
* cron for autoOpen and autoResolve chats in chatWoot

### you need to install this packages using go ###
* go install github.com/githubnemo/CompileDaemon      # autoreload app on change
* go install github.com/swaggo/swag/cmd/swag@latest   # install in the OS swag command
* go get -u github.com/gin-gonic/gin                  # framework
* go get -u github.com/joho/godotenv                  # cargar variables de .env
* go get -u github.com/jackc/pgx/v5                   # postgresql driver
* go get -u github.com/jackc/pgx/v5/pgxpool           # postgresql driver
* go get -u github.com/go-sql-driver/mysql            # mysql driver
* go get -u github.com/gin-contrib/i18n               # internacionalizacion de msjs
* go get -u github.com/staskobzar/goami2              # to connect to AMI in asterisk
* go get -u github.com/go-playground/validator/v10    # validadores forms
* go get -u gopkg.in/natefinch/lumberjack.v2          # logrotate
* go get -u github.com/go-co-op/gocron/v2             # crons
* go get -u github.com/swaggo/gin-swagger             # library to handle documentation on the project
* go get -u github.com/swaggo/files                   # library to handle documentation on the project

### you need also to create a .env file below are the related vars ### 

```
  PORT=7006

  # use release or debug mode, in debug: all request are logged with the header and body
  # also in debug mode all response from chatwoot api are store in log
  GIN_MODE=debug

  # mysql call_center variables
  DB_MYSQL=user:password|@tcp(ip_address:port)/database_name
  MYSQL_MAX_CONN=5
  MYSQL_MIN_CONN=2

  # postgres variables
  DB_POSTGRES=postgres://user:password@ip_address:port/database_name
  PGSQL_MAX_CONN=5
  PGSQL_MIN_CONN=1

  # chatwoot variables
  CHAT_TOKEN=api_access_token
  CHAT_TOGGLE_URL=http://server_url/api/v1/accounts/1/conversations/%d/toggle_status
  CHAT_NEWMSG_URL=http://server_url/api/v1/accounts/1/conversations/%d/messages

  # variables to handle basic auth for access to api documentation url is /docs/index.html
  DOC_USER=username_here
  DOC_PASSWD=password_here

  # variables to handle basic auth for access to api for grafana data_sources
  GRAFANA_USER=grafana
  GRAFANA_PASSWD=qwerty123**

  # variables to handle basic auth for access to apiRest for intranet
  APIREST_USER=apirest
  APIRESTPASSWD=qwerty123**

  # variables to use to connect to asterisk via AMI
  AMI_SERVER=ip_address:tcp_port
  AMI_USER=grafana
  AMI_PASSWD=*grafana*

```

### Example of job definition: in .crontab ###
#### must create .crontab file on root folder of project to operate cron jobs, checkout crontab_example.json ####
```
 .---------------- minute (0 - 59)
 |  .------------- hour (0 - 23)
 |  |  .---------- day of month (1 - 31)
 |  |  |  .------- month (1 - 12) OR jan,feb,mar,apr ...
 |  |  |  |  .---- day of week (0 - 6) (Sunday=0 or 7) OR sun,mon,tue,wed,thu,fri,sat
 |  |  |  |  |
 *  *  *  *  * 
```


### create service using systemctl on linux
#### create file /etc/systemd/system/ired_callcenter.service

```
[Unit]
Description=web service, to handle call center related tasks runs on port 7006

[Install]
WantedBy=multi-user.target

[Service]
Type=simple
User=root
Restart=always
WorkingDirectory=/usr/local/src/ired.com/callcenter
ExecStart=/usr/local/src/ired.com/callcenter/callcenter
```
