const { ServiceBroker } = require("moleculer");

let broker = new ServiceBroker({
  nodeID: "ws-connector-dev-cli",
  // transporter: "nats://localhost:4222",
  transporter: "nats://localhost:12008",
  logger: console,
  logLevel: "trace",
  requestTimeout: 3 * 1000,
});

broker.start()

// Start REPL
setTimeout(function () {
  broker.repl();
}, 1000)

/* some test command 
call ws-connector.metrics
call ws-online.onlineStatus --userID gotest-user-0
call ws-online.onlineStatusBulk --ids gotest-user-0,gotest-user-1
call ws-sender.send --ids gotest-user-0,gotest-user-1 --data.mid m123 --data.msg.a abc --data.msg.b 111 --data.msg.c true
*/
