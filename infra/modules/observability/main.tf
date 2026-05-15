locals {
  alarm_actions = var.alert_sns_topic_arn == null ? [] : [var.alert_sns_topic_arn]
}

resource "aws_cloudwatch_metric_alarm" "dlq_depth" {
  alarm_name          = "${var.name_prefix}-dlq-depth"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  alarm_description   = "DLQ contains at least one message."
  alarm_actions       = local.alarm_actions
  ok_actions          = local.alarm_actions
  dimensions          = { QueueName = var.dlq_name }
  treat_missing_data  = "notBreaching"
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "queue_age" {
  alarm_name          = "${var.name_prefix}-queue-age-gt-60s"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "ApproximateAgeOfOldestMessage"
  namespace           = "AWS/SQS"
  period              = 60
  statistic           = "Maximum"
  threshold           = 60
  alarm_description   = "Oldest ingest queue message is older than 60 seconds."
  alarm_actions       = local.alarm_actions
  ok_actions          = local.alarm_actions
  dimensions          = { QueueName = var.queue_name }
  treat_missing_data  = "notBreaching"
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "postgres_connections" {
  alarm_name          = "${var.name_prefix}-postgres-connections-80pct"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "DatabaseConnections"
  namespace           = "AWS/RDS"
  period              = 60
  statistic           = "Average"
  threshold           = var.postgres_max_connections * 0.8
  alarm_description   = "Postgres connection use is above 80% of the configured cap."
  alarm_actions       = local.alarm_actions
  ok_actions          = local.alarm_actions
  dimensions          = { DBInstanceIdentifier = var.rds_instance_id }
  treat_missing_data  = "notBreaching"
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "ingest_error_rate" {
  alarm_name          = "${var.name_prefix}-ingest-error-rate-gt-1pct"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 5
  threshold           = 1
  alarm_description   = "Ingest 5xx/error rate exceeds 1% over 5 minutes."
  alarm_actions       = local.alarm_actions
  ok_actions          = local.alarm_actions
  treat_missing_data  = "notBreaching"
  metric_query {
    id          = "error_rate"
    expression  = "100 * errors / MAX([requests, 1])"
    label       = "Ingest error rate percent"
    return_data = true
  }
  metric_query {
    id = "errors"
    metric {
      metric_name = "throne_ingest_requests_total"
      namespace   = "Throne/PoC"
      period      = 60
      stat        = "Sum"
      dimensions  = { code = "error" }
    }
  }
  metric_query {
    id = "requests"
    metric {
      metric_name = "throne_ingest_requests_total"
      namespace   = "Throne/PoC"
      period      = 60
      stat        = "Sum"
      dimensions  = { code = "all" }
    }
  }
  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "inference_p99" {
  alarm_name          = "${var.name_prefix}-inference-p99-gt-30s"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "throne_inference_duration_seconds_p99"
  namespace           = "Throne/PoC"
  period              = 60
  statistic           = "Maximum"
  threshold           = 30
  alarm_description   = "Inference p99 exceeds 30 seconds."
  alarm_actions       = local.alarm_actions
  ok_actions          = local.alarm_actions
  treat_missing_data  = "notBreaching"
  tags                = var.tags
}
