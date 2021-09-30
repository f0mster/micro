# Microservice

## Требования

* protoc
* https://github.com/planetscale/vtprotobuf

## Кодогенерация

Установка: `go install github.com/f0mster/micro/cmd/...`

Пример для файла: `api.proto`

Перед запуском выполнить
```protoc -I=. --proto_path=. --go_out=. --go-vtproto_out=. api.proto```
После чего выполнить
```micro-rpc-code-gen -proto=api.proto```

### Функции

Описать в proto rpc функции

```
service SessionInternalAPIService {
	rpc Connect (ConnectReq) returns (ConnectResp);
}
```

В результате кодогенерации будут созданы `client` и `server`. Клиент вызывает функции, сервер отвечает на вызов функций.

### События

В proto файле нужно создать message для события. После этого перед описанием сервиса в аннотации
написать `// @event message1, message2....`
события можно писать через запятую или в несколько строк

```
// @event OnConnect
// @event OnDisconnect
// @event OnUserOnline, OnUserOffline
/*
 * @event UserChanged
 */
service SessionInternalAPIService {
}
```

В результате кодогенерации в client появится функции `SubscribeOnConnect`, `SubscribeOnDisconnect`
, `SubscribeOnUserOnline`, `SubscribeOnUserOffline`, `SubscribeUserChanged`
В сервере появится функции `PublishOnConnect`, `PublishOnDisconnect`, `PublishOnUserOnline`, `PublishOnUserOffline`
, `PublishUserChanged`

## Использование

Примеры смотреть в директории `examples`