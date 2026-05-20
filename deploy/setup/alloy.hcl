loki.write "to_loki" {
  endpoint {
    url = "http://${env.MONITORING_SERVER}:3100/loki/api/v1/push"
  }
}

loki.source.journal "journal" {
  forward_to = [loki.write.to_loki.receiver]

  labels = {
    job = "systemd-journal",
  }
}

// =====================
// TRACES -> TEMPO
// =====================

otelcol.receiver.otlp "default" {
  http {
    endpoint = "0.0.0.0:4318"
  }
  grpc {
    endpoint = "0.0.0.0:4317"
  }

  output {
    traces = [otelcol.processor.batch.default.input]
    metrics = [otelcol.processor.batch.metrics.input]
    logs = [otelcol.processor.batch.logs.input]
  }
}

otelcol.processor.batch "default" {
  output {
    traces = [otelcol.exporter.otlp.to_tempo.input]
  }
}

otelcol.processor.batch "metrics" {
  output {
    metrics = [otelcol.exporter.prometheus.to_prometheus.input]
  }
}

otelcol.processor.batch "logs" {
  output {
    logs = [otelcol.exporter.loki.to_loki.input]
  }
}

otelcol.exporter.otlp "to_tempo" {
  client {
    endpoint = "${env.MONITORING_SERVER}:4317"
    tls {
      insecure = true
    }
  }
}

otelcol.exporter.prometheus "to_prometheus" {
  forward_to = [prometheus.scrape.default.receiver]
}

prometheus.scrape "default" {
  targets = prometheus.scrape.default.targets
  forward_to = [prometheus.remote_write.default.receiver]
  job_name = "otel-collected-metrics"
}

prometheus.remote_write "default" {
  endpoint {
    url = "http://${env.MONITORING_SERVER}:9090/api/v1/write"
  }
}

otelcol.exporter.loki "to_loki" {
  forward_to = [loki.write.to_loki.receiver]
}
