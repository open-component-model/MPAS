package main

deny[msg] {
  not input.replicas

  msg := "Replicas must be set"
}

deny[msg] {
  input.replicas > 2

  msg := "Replicas must be less than 2"
}

deny[msg] {
  input.replicas == 0

  msg := "Replicas must not be zero"
}

