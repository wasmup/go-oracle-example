# go-oracle-DEMO
Example Go code to use Oracle instant client

## Overview
This repository provides a comprehensive example of connecting to and interacting with an Oracle instant client/database using Go and the Godror driver.  


## Build Compiler image
```sh
# cd ./build_compiler_image/bin
# wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
# ls -lh
# cd ..

cd ./build_compiler_image

docker build -t oraclelinux-go:1.24.5 .

docker history oraclelinux-go:1.24.5

docker run --rm oraclelinux-go:1.24.5 go version
# go version go1.24.5 linux/amd64

docker images
# REPOSITORY                                    TAG                  IMAGE ID       CREATED        SIZE
# oraclelinux-go                                1.24.5               1856b78980cc   4 hours ago    1.12GB

cd ..
```


## Run Oracle DB 

```sh
docker pull container-registry.oracle.com/database/free:latest

# docker volume rm oracle-data --force
docker volume create oracle-data

docker run -d --name oracle1  --restart unless-stopped --network host -p 1521:1521 -e ORACLE_PASSWORD -v oracle-data:/ORCL container-registry.oracle.com/database/free:latest
docker ps
# docker start oracle1
# docker stop oracle1
# docker rm  oracle1

docker exec -it oracle1 sqlplus / as sysdba
# SQL*Plus: Release 23.0.0.0.0 - Production on Wed Jul 23 15:30:49 2025
# Version 23.8.0.25.04
```

## Create DB and user

```sql

CREATE PLUGGABLE DATABASE DEMO ADMIN USER admin2 IDENTIFIED BY password2 FILE_NAME_CONVERT = ('/opt/oracle/oradata/DB23AI/', '/opt/oracle/oradata/DEMO/', '/opt/oracle/oradata/FREE/pdbseed/', '/opt/oracle/oradata/DEMO/pdbseed/');

ALTER PLUGGABLE DATABASE DEMO OPEN;
ALTER SESSION SET CONTAINER = DEMO;
GRANT DBA TO admin2;

exit
```


## Build App image
```sh
go mod tidy -v
go mod vendor
ls -lh
docker build -t server:1.0.1 .

docker images
# REPOSITORY                                    TAG                  IMAGE ID       CREATED          SIZE
# server                                        1.0.1                4d786958890b   9 seconds ago    637MB

```

## Run App image
```sh
export DB_TIMEOUT="60s"
export DEMO_ORACLE_USER="admin2"
export DEMO_ORACLE_PASSWORD="password2"
export DEMO_ORACLE_SERVER="localhost:1521"
export DEMO_ORACLE_SERVICE_NAME="DEMO"


docker run -it  -e DEMO_ORACLE_USER   --rm server:1.0.1 /bin/bash 
ls -lh  /app/server
echo $LD_LIBRARY_PATH
echo $DEMO_ORACLE_USER
exit


docker run --rm --network host \
  -e DB_TIMEOUT \
  -e DEMO_ORACLE_USER \
  -e DEMO_ORACLE_PASSWORD \
  -e DEMO_ORACLE_SERVER \
  -e DEMO_ORACLE_SERVICE_NAME \
  server:1.0.1


```

Sample Output:
```
time=2025-07-23T16:19:22.780Z level=INFO source=server/main.go:31 msg=Go Version=go1.24.5 OS=linux ARCH=amd64 GOAMD64=v3 now=2025-07-23T16:19:22.780Z Local=UTC
admin2/password2@localhost:1521/DEMO
time=2025-07-23T16:19:22.859Z level=INFO source=server/main.go:66 msg=ping_ok

Oracle Database Version Information:
Oracle Database 23ai Free Release 23.0.0.0.0 - Develop, Learn, and Run for Free

Concise Instance Version: 23.0.0.0.0

Product Component Version:
Product: Oracle Database 23ai Free
Version: 23.0.0.0.0
Status: Develop, Learn, and Run for Free
Attempting to drop user "user[2]admin" if it exists...
User "user[2]admin" dropped successfully (if it existed).
Creating user "user[2]admin"...
User "user[2]admin" created successfully.
Granting roles and privileges to "user[2]admin"...
Roles and privileges granted to "user[2]admin" successfully.

New admin user '"user[2]admin"' with password '"pass[2]special_char"' created successfully.
```

## Set user name with special chars

```sh
export DB_TIMEOUT="60s"
export DEMO_ORACLE_USER='user[2]admin'
export DEMO_ORACLE_PASSWORD='pass[2]special_char'
export DEMO_ORACLE_SERVER="localhost:1521"
export DEMO_ORACLE_SERVICE_NAME="DEMO"

docker run --rm --network host \
  -e DB_TIMEOUT \
  -e DEMO_ORACLE_USER \
  -e DEMO_ORACLE_PASSWORD \
  -e DEMO_ORACLE_SERVER \
  -e DEMO_ORACLE_SERVICE_NAME \
  server:1.0.1


```

Sample Output2:
```
time=2025-07-23T16:47:11.895Z level=INFO source=server/main.go:31 msg=Go Version=go1.24.5 OS=linux ARCH=amd64 GOAMD64=v3 now=2025-07-23T16:47:11.895Z Local=UTC
"user[2]admin"/pass[2]special_char@localhost:1521/DEMO
time=2025-07-23T16:47:12.033Z level=INFO source=server/main.go:66 msg=ping_ok

Oracle Database Version Information:
Oracle Database 23ai Free Release 23.0.0.0.0 - Develop, Learn, and Run for Free

Concise Instance Version: 23.0.0.0.0

Product Component Version:
Product: Oracle Database 23ai Free
Version: 23.0.0.0.0
Status: Develop, Learn, and Run for Free
Attempting to drop user "user[2]admin" if it exists...
time=2025-07-23T16:47:12.219Z level=ERROR source=server/main.go:129 msg=drop_user_er error=newUsername "\"user[2]admin\""="ORA-01940: cannot drop a user who is currently connected\nHelp: https://docs.oracle.com/error-help/db/ora-01940/"

```


## License
MIT License
 
## References
- [Godror Documentation](https://github.com/godror/godror)
- [Go SQL Package](https://golang.org/pkg/database/sql/)
- [Oracle Database Documentation](https://docs.oracle.com/en/database/)

