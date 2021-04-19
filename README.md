# Microservice

## Требования

* protoc
* https://github.com/gogo/protobuf

## Кодогенерация

Установка: вызывать один из предложенных вариантов в `Makefile (linux, win, darwin)`

Пример для файла: `api.proto`

Перед запуском выполнить
```protoc -I=. --proto_path=. --gofast_out=. api.proto```
После чего выполнить
```rendall-rpc-code-gen -proto=api.proto```

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