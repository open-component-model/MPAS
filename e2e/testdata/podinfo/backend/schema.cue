// # Configuration Instructions

// This is the api service of the podinfo microservices application.

// The following parameters are available for configuration:

// | Parameter | Type    | Default          | Description                            |
// |-----------|---------|------------------|----------------------------------------|
// | replicas  | integer | 2                | Number of replicas for the application |
// | cacheAddr | string  | tcp://redis:6379 | Address of the cache server            |

#SchemaVersion: "v1.0.0"

// an object named after the target resource is expected in order to pick parameters from it.
backend: {
  // this field has a default value of 2 and must be less than 4
  replicas: *2 | int & >0 & <4
  // this field is required and must match the regex
  cacheAddr: *"tcp://redis:6379" | string & =~"^tcp://[a-z.-]+:\\d+"
}
