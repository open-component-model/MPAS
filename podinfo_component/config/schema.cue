//  # Configuration Instructions
//
//  This is the api service of the podinfo microservices application.
//
//  The following parameters are available for configuration:
//
//  | Parameter           | Type    | Default          | Description                            |
//  |---------------------|---------|------------------|----------------------------------------|
//  | replicas            | integer | 2                | Number of replicas for the application |
//  | message             | string  | "Hello from Go!" | Message to display                     |
//  | serviceAccountName  | string  | ""               | Name of the service account to use     |

#SchemaVersion: "v1.0.0"

// an object named after the target resource is expected in order to pick parameters from it.
podinfo: {
  // this field has a default value of 2
  replicas: *2 | int
  // this field is optional
  message?: string
  // this field is optional
  serviceAccountName?: string
}

