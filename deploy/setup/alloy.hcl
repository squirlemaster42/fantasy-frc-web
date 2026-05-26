loki.write "to_loki" {
  endpoint {
    url = "http://<monitoring_server>:3100/loki/api/v1/push"
  }
}

loki.source.journal "journal" {
  forward_to = [loki.write.to_loki.receiver]

  labels = {
    job = "systemd-journal",
  }
}

// =====================
// TRACES → TEMPO
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
  }
}

otelcol.processor.batch "default" {
  output {
    traces = [otelcol.exporter.otlp.to_tempo.input]
  }
}

otelcol.exporter.otlp "to_tempo" {
  client {
    endpoint = "<monitoring_server>:4317"
    tls {
      insecure = true
    }
  }
}
