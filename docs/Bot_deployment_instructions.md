### Installing the database in a docker container

``` shell
docker run -d \
    --name postgres-db \
    -e POSTGRES_USER=admin \
    -e POSTGRES_PASSWORD=admin \
    -e POSTGRES_DB=botdb \
    -p 5432:5432 \
    -v postgres_data:/var/lib/postgresql/data \
    postgres:16

migrate -path ./migrations -database "postgres://admin:admin@localhost:5432/botdb?sslmode=disable" up
```

url db - `postgresql://botuser:botpass@localhost:5432/botdb`

Remove All Data (optional)
```shell
docker stop postgres-db
docker rm postgres-db
docker volume rm postgres_data
```

Remove Docker Image (optional)
```shell
docker rmi postgres:16
```
