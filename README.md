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

message OnConnect {
    string id = 1;
}
message OnDisconnect {}
message OnUserOnline {}
message OnUserOffline {}
message UserChanged {
    string id = 1;
    string changed_field = 2;
}
```

В результате кодогенерации в client появится функции `SubscribeOnConnect`, `SubscribeOnDisconnect`
, `SubscribeOnUserOnline`, `SubscribeOnUserOffline`, `SubscribeUserChanged`
В сервере появится функции `PublishOnConnect`, `PublishOnDisconnect`, `PublishOnUserOnline`, `PublishOnUserOffline`
, `PublishUserChanged`

### Custom topics for events

If you need to set custom topic for event you should add `:your_topic_name` after message name. 
```
// @event OnUserOnline:my.awesome.topic, OnUserOffline:another.topic
service SessionInternalAPIService {
}
message OnUserOnline {}
message OnUserOffline {}
```


## Использование

Примеры смотреть в директории `examples`