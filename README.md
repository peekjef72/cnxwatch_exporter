# connection watch exporter
Exporter of socket connection status for prometheus.

The exporter generates labeled metrics with the status of the socket connection equivalent to execute:
```
netstat -nap | grep
```
## Why a connection watch exporter?
Most of the time applications interact each other using sockets. It may not enought to know that the underlying processes are running it may be usefull to detect they have established connections to machines on specified ports.
This exporter reports the status and number of connections according to the configuration.

## Getting started
To run it:
```
./cnxwatch_exporter
```
by default the exporter will read the configuration file *config/config.yml*. Another file can be passed as a flag:
```
./cnxwatch_exporter --config-file=config/user_config.yml
```

The metrics are available at http://localhost:9293/metrics. Here is an example: 
```
# HELP connection_status_count number of socket with same parameter.
# TYPE connection_status_count gauge
connection_status_count{dsthost="*",dstport="*",name="hostname-grafana",process="*",protocol="tcp6",srchost="::",srcport="3000",status="listen"} 1
connection_status_count{deshost="127.0.0.1",dstport="22",name="ssh-from-localhost",process="*",protocol="tcp",srchost="127.0.0.1",srcport="*",status="established"} 0
# HELP connection_status_up Connection status of the socket (0 down - 1 up).
# TYPE connection_status_up gauge
connection_status_up{deshost="*",dstport="*",name="hostname-grafana",process="*",protocol="tcp6",srchost="::",srcport="3000",status="listen"} 1
connection_status_up{deshost="127.0.0.1",dstport="22",name="ssh-from-localhost",process="*",protocol="tcp",srchost="127.0.0.1",srcport="*",status="established"} 0
```
The metrics are:
* **connection_status_up** with the labels for each socket iin the config and the following possible values:
  - 1: Connection found
  * 0: Connection not found

* **connection_status_count** with the labels for each socket counting the number of connection with the parameters.

## Usage
To configure the sockets that te exporter will check, a yaml configuration file is used. This is a configuration example:

```
sockets:
  - name: hostname-grafana 
    srcHost: "::"
    port: 3000
    status: listen
    protocol: tcp6
  - name: ssh-from-localhost
    srcHost: 127.0.0.1
    dstHost: 127.0.0.1
    dstPort: 22
    status: established
```
The fields of the sockets to configure are:
* **name**: A name to be able to filter by this in prometheus
* **host** or **srcHost**: source Hostname or IP of the socket
* **port** or **srcPort**: source Port to check. Default empty meaning not checked
* **dstHost**: destination Hostname or IP of the socket. Default empty meaning not checked
* **dstPort**: destination Port to check. Default empty meaning not checked
* **protocol**: network parameter. Known networks are: "tcp" (IPv4-only), "tcp6" (IPv6-only), "udp" (IPv4-only), "udp6" (IPv6-only. If not defined, it will be set to "tcp" by default. 
* **processName**: the process owner of the socket; **WARNING** collected only if root or owner the socket !
* **status**: the status of the socket: Should be "listen" or "established".

The following fields will be used as labels in the metric:
* name
* srchost
* srcport
* dsthost
* dstport
* protocol
* process
* status

## Contributing
Please read the [CONTRIBUTING](https://github.com/peekjef72/cnxwatch_exporter/blob/master/CONTRIBUTING.md) guidelines.

## Credits
- [peekjef72](https://github.com/peekjef72) - *Initial work*

## License
This project is published under Apache 2.0, see [LICENSE](https://github.com/peekjef72/cnxwatch_exporter/blob/master/LICENSE).
