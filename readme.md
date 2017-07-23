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

## Limitations

At the moment all messages of all Slack channel the bot is a member of get streamed to "topic4channel", including "direct" messages

## TODO

* Filter Channel (see Limitations)
* Support for direct messages
1. pipe them to topic
1. reply from topic (how?)
* Optimize resolving of usernames