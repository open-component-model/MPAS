package main

deny[msg] {
  not input.message

  msg := "Message is required"
}
deny[msg] {
  allowed_colors = ["red", "blue", "green", "yellow"]

  not input.color

  msg:= "color is required"
}

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
