# MySQL User API

Go 기반의 간단한 User REST API와 MySQL 데이터베이스로 구성된 Kubernetes 테스트용 프로젝트입니다.

## 아키텍처

```text
Client
  │
  │ HTTP :8080
  ▼
MySQL User API
  │
  │ MySQL :3306
  ▼
MySQL Database
```

API 서버와 MySQL 데이터베이스를 같은 Pod 또는 서로 다른 Pod에 배치하여 Kubernetes의 컨테이너 및 네트워크 구조를 테스트할 수 있습니다.

## 프로젝트 구조

```text
.
├── api/
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── main.go
│
├── db/
│   ├── Dockerfile
│   └── init.sql
│
├── k8s/
│   ├── api.yaml
│   ├── db.yaml
│   └── secret.yaml
│
└── README.md
```

---

# MySQL User API 서버

## 디렉터리

```text
/api
```

## 포트

```text
8080
```

## Docker 이미지

```text
nochank/mysql-api:v2
```

## API

### 사용자 단건 조회

```http
GET /user/{user_id}
```

### 전체 사용자 조회

```http
GET /users
```

### 사용자 생성

```http
POST /user
Content-Type: application/json
```

요청 Body:

```json
{
  "name": "string",
  "email": "string"
}
```

## 환경변수

| 환경변수 | 기본값 | 설명 |
| --- | --- | --- |
| `MYSQL_ROOT_PASSWORD` | `password` | MySQL 접속 비밀번호 |
| `MYSQL_DATABASE` | `db` | 사용할 데이터베이스 이름 |
| `MYSQL_HOST` | `127.0.0.1` | MySQL 서버 주소 |
| `MYSQL_PORT` | `3306` | MySQL 서버 포트 |
| `MYSQL_USER` | `root` | MySQL 접속 사용자 |

환경변수가 전달되지 않은 경우 API 서버는 위의 기본값을 사용합니다.

## MYSQL_HOST 설정

`MYSQL_HOST`는 API 서버와 MySQL 데이터베이스의 배치 방식에 따라 변경해야 합니다.

### 같은 Pod에 배치하는 경우

API 컨테이너와 MySQL 컨테이너가 같은 Pod에 위치하면 **동일한 네트워크 네임스페이스를 공유**합니다.

따라서 API 컨테이너에서 `127.0.0.1`을 사용하여 MySQL 컨테이너에 접근할 수 있습니다.

```text
MYSQL_HOST=127.0.0.1
```

구조:

```text
Pod
│
├── API 컨테이너 :8080
│      │
│      └── 127.0.0.1:3306
│
└── MySQL 컨테이너 :3306
```

같은 Pod의 컨테이너들은 동일한 Pod IP를 공유하기 때문에 `localhost` 또는 `127.0.0.1`을 사용하여 서로 통신할 수 있습니다.

### 서로 다른 Pod에 배치하는 경우

API와 MySQL을 각각 별도의 Pod에 배치하면 `127.0.0.1`을 사용할 수 없습니다.

각 Pod가 서로 다른 Pod IP를 가지기 때문입니다.

이 경우 MySQL Pod를 대상으로 하는 Kubernetes Service를 생성하고 API 서버에서 해당 **Service 이름을 Host로 사용**합니다.

예를 들어 MySQL Service 이름이 `db-service`인 경우:

```text
MYSQL_HOST=db-service
```

구조:

```text
API Pod
   │
   │ db-service:3306
   ▼
db-service
   │
   ▼
MySQL Pod :3306
```

Kubernetes의 CoreDNS가 `db-service`라는 이름을 Service의 ClusterIP로 해석하고, Service가 요청을 대상 MySQL Pod의 `3306` 포트로 전달합니다.

API Deployment에서는 다음과 같이 설정할 수 있습니다.

```yaml
env:
  - name: MYSQL_HOST
    value: "db-service"
```

배치 방식에 따른 `MYSQL_HOST` 설정은 다음과 같습니다.

| 배치 방식 | `MYSQL_HOST` |
| --- | --- |
| API와 MySQL이 같은 Pod | `127.0.0.1` |
| API와 MySQL이 서로 다른 Pod | `db-service` |

---

# MySQL 데이터베이스

## 디렉터리

```text
/db
```

## 포트

```text
3306
```

## Docker 이미지

```text
nochank/mysql:v2
```

## 환경변수

| 환경변수 | 기본값 | 설명 |
| --- | --- | --- |
| `MYSQL_ROOT_PASSWORD` | `password` | MySQL root 계정 비밀번호 |
| `MYSQL_DATABASE` | `db` | 생성할 데이터베이스 이름 |

## 데이터베이스 초기화

MySQL Docker 이미지에는 다음 경로에 `init.sql`이 포함되어 있습니다.

```text
/docker-entrypoint-initdb.d/init.sql
```

`init.sql`에서는 `db` 데이터베이스와 `users` 테이블을 초기화합니다.

```sql
CREATE DATABASE IF NOT EXISTS db;

USE db;

CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL
);
```

MySQL 데이터 디렉터리가 처음 초기화될 때 `/docker-entrypoint-initdb.d/init.sql`이 실행됩니다.

Kubernetes에서 `/var/lib/mysql`에 PVC를 마운트한 경우 최초 실행 시 데이터베이스가 PVC에 초기화됩니다.

```text
MySQL Pod 최초 실행
        │
        ▼
PVC → /var/lib/mysql
        │
        ▼
MySQL 데이터 없음
        │
        ▼
MySQL 초기화
        │
        ▼
init.sql 실행
        │
        ▼
db / users 테이블 생성
        │
        ▼
실제 데이터 → PVC에 저장
```

이후 Pod가 삭제되고 새 Pod가 동일한 PVC를 다시 마운트하면 기존 MySQL 데이터를 그대로 사용합니다.

```text
새 MySQL Pod
     │
     ▼
기존 PVC 마운트
     │
     ▼
/var/lib/mysql에 기존 데이터 존재
     │
     ▼
init.sql 재실행 안 함
     │
     ▼
기존 데이터 사용
```

따라서 역할은 다음과 같이 구분됩니다.

```text
Docker Image
├── MySQL 8.4
└── init.sql
      │
      └── 최초 데이터베이스 초기화

PVC
└── /var/lib/mysql
      │
      └── 실제 MySQL 데이터 영구 저장
```

---

# Docker 이미지

## API 이미지 다운로드

```bash
docker pull nochank/mysql-api:v2
```

## MySQL 이미지 다운로드

```bash
docker pull nochank/mysql:v2
```

## MySQL 컨테이너 실행

```bash
docker run -d \
  --name mysql \
  -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=db \
  nochank/mysql:v2
```

---

# Kubernetes 구성

Kubernetes 테스트를 위한 Manifest 파일은 `k8s` 디렉터리에서 관리합니다.

```text
k8s/
├── api.yaml
├── db.yaml
└── secret.yaml
```

각 Manifest의 역할은 다음과 같습니다.

| 파일 | 역할 |
| --- | --- |
| `api.yaml` | User API Deployment와 Service 구성 |
| `db.yaml` | MySQL Deployment와 Service 구성 |
| `secret.yaml` | MySQL 데이터베이스 이름과 비밀번호 관리 |

## API 구성

`api.yaml`은 `app: api` 라벨을 가진 API Pod를 생성하는 Deployment와 해당 Pod에 접근하기 위한 `api-service`를 정의합니다.

```text
api-service :80
      │
      │ selector: app=api
      ▼
API Pod :8080
      │
      │ MYSQL_HOST=db-service
      ▼
db-service :3306
```

API Pod는 MySQL과 별도의 Pod에서 실행되므로 다음 설정을 사용하여 DB에 접근합니다.

```text
MYSQL_HOST=db-service
MYSQL_PORT=3306
```

## DB 구성

`db.yaml`은 `app: db` 라벨을 가진 MySQL Pod를 생성하는 Deployment와 해당 Pod에 접근하기 위한 `db-service`를 정의합니다.

```text
db-service :3306
      │
      │ selector: app=db
      ▼
MySQL Pod :3306
      │
      ▼
MySQL Database
```

API Pod에서 `db-service:3306`으로 연결하면 CoreDNS가 `db-service`를 Service의 ClusterIP로 해석합니다.

Service는 `app: db` 라벨을 가진 MySQL Pod를 대상으로 요청을 전달합니다.

## Secret 구성

`secret.yaml`에서는 API와 MySQL에서 공통으로 사용하는 데이터베이스 설정을 관리합니다.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
stringData:
  db-password: "password"
  db-name: "db"
```

API와 DB Deployment에서는 `secretKeyRef`를 사용하여 Secret 값을 환경변수로 전달받습니다.

```text
                  db-secret
                  /       \
                 ▼         ▼
              API Pod   MySQL Pod
```

`secret.yaml`에 포함된 `password`는 Kubernetes Secret 사용 방법을 보여주기 위한 실습용 예제 값입니다.

## 전체 Kubernetes 구조

```text
                     Kubernetes Cluster

             ┌───────────────────────────────┐
             │                               │
             │           db-secret           │
             │           /      \            │
             │          ▼        ▼           │
             │                               │
Client ──────┼──→ api-service :80            │
             │          │                    │
             │          │ app=api            │
             │          ▼                    │
             │     API Pod :8080             │
             │          │                    │
             │          │ db-service:3306    │
             │          ▼                    │
             │     db-service :3306          │
             │          │                    │
             │          │ app=db             │
             │          ▼                    │
             │     MySQL Pod :3306           │
             │                               │
             └───────────────────────────────┘
```

API Pod에서 사용하는 환경변수는 다음과 같습니다.

```text
MYSQL_USER=root
MYSQL_ROOT_PASSWORD=password
MYSQL_DATABASE=db
MYSQL_HOST=db-service
MYSQL_PORT=3306
```

MySQL Pod에서는 다음 환경변수를 사용합니다.

```text
MYSQL_ROOT_PASSWORD=password
MYSQL_DATABASE=db
```

---

# Kubernetes Manifest

## secret.yaml

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
stringData:
  db-password: "password"
  db-name: "db"
```

## db.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: db-deploy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: db
  template:
    metadata:
      labels:
        app: db
    spec:
      containers:
        - name: db
          image: nochank/mysql:v2
          ports:
            - containerPort: 3306
          env:
            - name: MYSQL_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: db-password

            - name: MYSQL_DATABASE
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: db-name

---
apiVersion: v1
kind: Service
metadata:
  name: db-service
spec:
  selector:
    app: db
  ports:
    - port: 3306
      targetPort: 3306
  type: ClusterIP
```

## api.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-deploy
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
        - name: api
          image: nochank/mysql-api:v2
          ports:
            - containerPort: 8080
          env:
            - name: MYSQL_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: db-password

            - name: MYSQL_DATABASE
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: db-name

            - name: MYSQL_HOST
              value: "db-service"

            - name: MYSQL_PORT
              value: "3306"

            - name: MYSQL_USER
              value: "root"

---
apiVersion: v1
kind: Service
metadata:
  name: api-service
spec:
  selector:
    app: api
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
```

---

# Kubernetes 배포

프로젝트 루트에서 다음 명령어를 실행하여 모든 Manifest를 적용할 수 있습니다.

```bash
kubectl apply -f ./k8s/
```

리소스를 확인합니다.

```bash
kubectl get deployments
kubectl get pods
kubectl get services
```

각 Service와 Pod의 연결 관계는 다음과 같습니다.

```text
api-service
selector: app=api
      │
      ├── API Pod
      └── API Pod


db-service
selector: app=db
      │
      └── MySQL Pod
```

API 애플리케이션에는 MySQL 연결 재시도 로직이 포함되어 있으므로 API와 DB 리소스가 동시에 생성되더라도 MySQL이 준비될 때까지 연결을 재시도합니다.

따라서 `kubectl apply -f ./k8s/` 실행 시 API Deployment와 DB Deployment의 생성 순서에 의존하지 않도록 구성할 수 있습니다.