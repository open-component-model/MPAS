// # Configuration Instructions

// This is the cache for the podinfo microservices application.

// The following parameters are available for configuration:

// | Parameter | Type | Default | Description                          |
// |-----------|------|---------|--------------------------------------|
// | replicas  | int  | 1       | The number of replicas for the cache |

#SchemaVersion: "v1.0.0"

// an object named after the target resource is expected in order to pick parameters from it.
redis: {
  // this field has a default value of 1 and must equal to 1
  replicas: *1 | 1
}
