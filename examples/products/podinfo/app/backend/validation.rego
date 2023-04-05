package main

deny[msg] {
  not input.replicas

  msg := "Replicas must be set"
}

deny[msg] {
  input.replicas > 4

  msg := "Replicas must be less than 4"
}

deny[msg] {
  input.replicas == 0

  msg := "Replicas must not be zero"
}

deny[msg] {
  not input.message

  msg := "Message is required"
}
