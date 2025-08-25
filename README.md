# prometheus-nginx-log-exporter

An nginx logs to prometheus exporter

## Development

1. init repo

   ```sh
   go mod tidy
   ```

1. prepare config and log file

   ```sh
   cp ./conf/access.log.example ./conf/access.log
   cp ./conf/nginx-log-exporter.yaml.example ./conf/nginx-log-exporter.yaml

   # update the absolute path in the config file for the log file
   ```

1. run the app

   ```sh
   go run . --config-file ./conf/nginx-log-exporter.yaml

   # to simulate the log generation, you can manually copy new lines to the log file
   ```
