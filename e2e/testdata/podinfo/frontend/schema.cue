// # Configuration Instructions

// This is the web frontend for the podinfo microservices application.

// The following parameters are available for configuration:

// | Parameter | Type    | Default       | Description                                |
// |-----------|---------|---------------|--------------------------------------------|
// | color     | string  | red           | The background color of the website        |
// | message   | string  | Hello, world! | The message to display                     |
// | replicas  | integer | 1             | The number of replicas for the application |

#SchemaVersion: "v1.0.0"

// an object named after the target resource is expected in order to pick parameters from it.
frontend: {
  color: *"red" | "blue" | "green" | "yellow"
  // this field is required
  message !: *"Hello, world!" | string
  // this field has a default value of 1 and must be an integer between 1 and 2
  replicas: *1 | int & >0 & <2
}
