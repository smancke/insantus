
insantus
============

A very simple health check and statuspage app.

Features
-------------
* Http checks
* SCP checks
* Certificate checks
* Multi Environment
* Notifications by email

Configuration
--------------
Adjust the example files matching your requirements:
* checks.yml
* environments.yml

See the `config.go` for details about the available options.


Run it using go
-----------------

```
go get github.com/smancke/insantus
insantus --help
```

Run it with docker
---------------------
To try it out:
```
docker run --rm -p 80:80 smancke/insantus
```

With your configuration:
```
docker run -v /your/checks.yml:/checks.yml \
           -v /your/environments.yml:/environments.yml \
           -v /path/to/presist:/data \
           -p 80:80 smancke/insantus \
```
