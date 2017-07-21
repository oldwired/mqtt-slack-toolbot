# Readme
Bot for Slack/MQTT Coupling

## Function
All messages posted to a specific slack channel get published to a specified MQTT topic.
All messages to specified MQTT topics get posted to the slack channel.

## Sample Config
`config.json`
```javascript
{
    "secret": "REDACTED",
    "channel": "#testing",
    "topics": ["test/test", "test/test2"],
    "topic4channel": "test/chat",
    "botStatusTopic": "test/chat/toolbot",
    "debug": false,
    "broker": "127.0.0.1",
    "port": "1883",
    "clientID": "mqttbot",
    "enterMessage": "Sit right there; make yourself comfortable. Remember the time we used to spend playing chess together?"
}
```

