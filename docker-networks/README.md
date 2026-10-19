# Домашнее задание: Docker-сети

Отчёт о трёх практических заданиях: стандартная и пользовательская bridge-сети,
изоляция сетей, режимы `host` и `none`, схема Nginx → API → MySQL.

Среда: Windows, PowerShell, Docker Desktop, Linux-контейнеры.
Команды ниже предназначены для последовательного выполнения по разделам.
Результаты приведены по выполненным проверкам; вывод местами сокращён.
IP-адреса относятся к конкретным запускам и могут изменяться.

Перед началом:

```powershell
docker version
```

## 1. Стандартная bridge и пользовательская bridge

### Стандартная сеть

```powershell
docker run -d --name c1 alpine sleep 3600
docker run -d --name c2 alpine sleep 3600
docker exec c1 ping -c 2 c2
```

Результат обращения по имени:

```text
ping: bad address 'c2'
```

Проверка по IP, полученному через `docker inspect`:

```powershell
$c2Ip = docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' c2
docker exec c1 ping -c 2 $c2Ip
```

Результат:

```text
PING 172.17.0.3 (172.17.0.3): 56 data bytes
64 bytes from 172.17.0.3: seq=0 ttl=64 time=0.133 ms
64 bytes from 172.17.0.3: seq=1 ttl=64 time=0.067 ms
```

### Пользовательская сеть

```powershell
docker rm -f c1 c2
docker network create mynet
docker run -d --name c1 --network mynet alpine sleep 3600
docker run -d --name c2 --network mynet alpine sleep 3600
docker exec c1 ping -c 2 c2
docker exec c1 cat /etc/resolv.conf
```

Результаты:

```text
PING c2 (172.25.0.3): 56 data bytes
64 bytes from 172.25.0.3: seq=0 ttl=64 time=0.175 ms
64 bytes from 172.25.0.3: seq=1 ttl=64 time=0.089 ms

nameserver 127.0.0.11
```

### Ответы на вопросы

- Без `--network` контейнер подключается к стандартной сети `bridge`.
  В ней нет автоматического разрешения имён контейнеров, но связь по IP работает.
- В пользовательской bridge-сети Docker автоматически разрешает имена контейнеров
  этой сети. Поэтому `ping c2` работает.
- Встроенный DNS Docker доступен в контейнере по адресу `127.0.0.11`.
- Стандартная bridge подходит для простых отдельных запусков. Для связанных
  сервисов удобнее отдельная пользовательская сеть: есть DNS по именам и можно
  отделить контейнеры одного проекта от других.

## 2. Изоляция сетей

```powershell
docker network create frontend
docker network create backend
docker run -d --name web --network frontend nginx:alpine
docker run -d --name db --network backend alpine sleep 3600
docker exec web ping -c 2 db
docker exec db ping -c 2 web
```

До подключения к общей сети:

```text
ping: bad address 'db'
ping: bad address 'web'
```

Подключение работающего контейнера ко второй сети:

```powershell
docker network connect backend web
docker exec web ping -c 2 db
docker inspect web
```

После подключения пинг успешен:

```text
PING db (172.27.0.2): 56 data bytes
64 bytes from 172.27.0.2: seq=0 ttl=64 time=0.477 ms
64 bytes from 172.27.0.2: seq=1 ttl=64 time=0.075 ms
```

В `NetworkSettings.Networks` у `web` два подключения:

| Сеть | IP контейнера web |
|---|---|
| backend | 172.27.0.3 |
| frontend | 172.26.0.2 |

### Ответы на вопросы

- Одинаковый тип `bridge` не означает, что сети общие. `frontend` и `backend` —
  отдельные сети, между которыми Docker по умолчанию ограничивает прямой доступ.
  До подключения общей сети контейнеры также не разрешают имена друг друга.
- `docker network connect backend web` добавляет подключение `web` к `backend`,
  сохраняя `frontend`. У контейнера теперь два IPv4-адреса, по одному в каждой сети.
- Два подключения нужны, например, API: через frontend он принимает запросы,
  через backend обращается к базе. Остальные контейнеры сетей от этого
  автоматически не получают доступ друг к другу.
- Для отсутствия внешней сети используется `--network none`. Остаётся только
  loopback — связь контейнера с самим собой.

Ошибка DNS сама по себе не доказывает блокировку прямого доступа по IP.
Отдельная проверка по IP выполнена в части 3C.

## 3. Host, none и практический сценарий

### A. Host-сеть

В Docker Desktop режим требует версии 4.34 или новее и включения
**Settings → Resources → Network → Enable host networking → Apply and restart**.
См. [документацию Docker](https://docs.docker.com/engine/network/drivers/host/#docker-desktop).
В Desktop эта возможность работает для TCP/UDP; её реализация отличается
от общего сетевого пространства имён на обычном Docker Engine под Linux.

```powershell
docker run -d --name host-web --network host nginx:alpine
docker port host-web
```

Первый запуск завершился ошибкой:

```text
bind() to 0.0.0.0:80 failed (98: Address in use)
```

Проверки `docker ps -a` и `docker logs --tail 30 host-web` показали, что Nginx
завершился. Командой `docker ps` найден контейнер `k8s-course-control-plane`,
публикующий `0.0.0.0:80->80/tcp`.

Для упражнения этот узел учебного Kubernetes временно остановлен:

```powershell
docker stop k8s-course-control-plane
docker start host-web
docker port host-web
```

После освобождения порта страница Nginx открылась по `http://localhost:80`.
Вывод `docker port host-web` пустой: публикация через `-p` не используется.
Остановка узла Kubernetes здесь нужна только из-за обнаруженного конфликта;
это не обязательный шаг для других окружений.

После проверки Nginx можно остановить, а узел Kubernetes вернуть:

```powershell
docker stop host-web
docker start k8s-course-control-plane
```

### B. None-сеть

```powershell
docker run -d --name isolated --network none alpine sleep 3600
docker exec isolated ip addr
docker exec isolated ping -c 2 -W 2 8.8.8.8
```

Внутри только интерфейс `lo`, с адресами `127.0.0.1` и `::1`.
При попытке обратиться наружу:

```text
PING 8.8.8.8 (8.8.8.8): 56 data bytes
ping: sendto: Network unreachable
```

### C. Nginx, API и MySQL

Целевая схема:

```text
Хост:8080 → Nginx:80 ── frontend ── API ── backend ── MySQL:3306
```

Очистка ресурсов предыдущих частей перед созданием схемы:

```powershell
docker rm -f c1 c2 web db host-web isolated
docker network rm mynet frontend backend
```

Команды затрагивают только перечисленные учебные ресурсы. Контейнеры и сети
других проектов удалять не требуется.

Создание схемы:

```powershell
docker network create frontend
docker network create backend
docker run -d --name mysql --network backend -e MYSQL_ROOT_PASSWORD=study123 mysql:8.4
docker run -d --name api --network backend alpine sleep 3600
docker network connect frontend api
docker run -d --name web --network frontend -p 8080:80 nginx:alpine
```

`study123` — демонстрационный пароль только для этой локальной практики.
Порты MySQL и API на хосте не публикуются.
Порт `8080` хоста должен быть свободен.

При первом запуске MySQL без пароля контейнер создался, но сервер завершился:

```text
Database is uninitialized and password option is not specified
```

Контейнер пересоздан с `-e MYSQL_ROOT_PASSWORD=study123`.
Также встретился конфликт имени `web`: остановленный контейнер предыдущего
упражнения сохранил имя. Он удалён перед созданием нового Nginx.

Страница Nginx открылась по `http://localhost:8080`.
Проверки сетевой связи:

```powershell
docker exec web ping -c 2 api
docker exec api ping -c 2 mysql
docker exec web ping -c 2 mysql
$mysqlIp = docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' mysql
docker exec web ping -c 2 -W 2 $mysqlIp
```

| Проверка | Результат |
|---|---|
| Nginx → API по имени | Ответы получены |
| API → MySQL по имени | 2 отправлено, 2 получено, 0% потерь |
| Nginx → MySQL по имени | `ping: bad address 'mysql'` |
| Nginx → MySQL по IP `172.26.0.2` | 2 отправлено, 0 получено, 100% потерь |

Общая сеть позволяет API обращаться к базе. У Nginx общей сети с MySQL нет:
имя базы не разрешается, а ICMP-проверка по IP также не получает ответов.
Здесь проверена сетевая связность. Проверка SQL-запросов и полноценное
проксирование HTTP в API в упражнение не входят: API — контейнер Alpine
с `sleep`, а не работающий HTTP-сервис.

### Ответы на вопросы

- **Host:** контейнер использует сеть Docker-хоста без публикации портов через
  `-p`. На Linux это позволяет обойти NAT и может быть полезно при работе с
  большим количеством портов. Минусы — меньше сетевой изоляции и конфликты
  портов. В упражнении конфликт порта 80 воспроизведён на практике.
- **None:** внешней сети нет, но loopback остаётся. Пример — обработка
  файлов, доступных контейнеру, без необходимости скачивать или отправлять данные.
  Отсутствие `--network` не равно `none`: по умолчанию используется bridge.
- **Почему Nginx не видит MySQL:** Nginx подключён только к frontend, MySQL —
  только к backend. Прямой доступ между ними не нужен: Nginx принимает HTTP,
  а API выполняет работу с данными. Разделение ограничивает лишний сетевой доступ.
  При этом Docker-хост остаётся доверенным: схема не изолирует базу от
  администратора хоста, а API по-прежнему требует собственной защиты.
- **React, Python API, PostgreSQL:** веб-сервер со сборкой React подключается
  к frontend, API — к frontend и backend, PostgreSQL — только к backend.
  Веб-сервер может передавать запросы `/api` в Python API, а тот обращается
  к PostgreSQL. JavaScript React выполняется в браузере, который не разрешает
  внутренние Docker-имена. Поэтому браузер использует публичный адрес
  веб-сервера с маршрутом `/api`, а веб-сервер обращается к API по Docker-сети.
  Порт базы на хосте не публикуется.
