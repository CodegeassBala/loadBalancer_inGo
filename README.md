# loadBalancer_inGo

A simple Load Balancer built in GoLang.

Provide a config.yml file which lists your servers' url to be hit and basic configurations like, Port on which your loadBalancer would run, max_no_of_connections for each client's connection pool and timeout for each request.

Exmple config File..

```yml
port: 8080
servers:
  - http://127.0.0.1:8001
  - http://127.0.0.1:8002

max_connections: 10
timeout: 10s
```

Steps to run the project:
  - go build
  - ./m


Happy Coding!!!
