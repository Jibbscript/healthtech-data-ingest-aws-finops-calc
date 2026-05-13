output "alarm_names" {
  value = {
    dlq_depth            = aws_cloudwatch_metric_alarm.dlq_depth.alarm_name
    queue_age            = aws_cloudwatch_metric_alarm.queue_age.alarm_name
    postgres_connections = aws_cloudwatch_metric_alarm.postgres_connections.alarm_name
    ingest_error_rate    = aws_cloudwatch_metric_alarm.ingest_error_rate.alarm_name
    inference_p99        = aws_cloudwatch_metric_alarm.inference_p99.alarm_name
  }
}
