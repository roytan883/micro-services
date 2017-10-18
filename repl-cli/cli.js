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
