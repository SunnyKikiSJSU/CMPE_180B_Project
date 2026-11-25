variable "project_name" {
  description = "Prefix for the budget name."
  type        = string
  default     = "cmpe180b-team-project"
}

variable "monthly_limit_usd" {
  description = "Monthly cost limit in US dollars for the whole account."
  type        = number
  default     = 20
}

variable "alert_emails" {
  description = "Addresses that get the budget alerts."
  type        = list(string)

  validation {
    condition     = length(var.alert_emails) > 0 && alltrue([for e in var.alert_emails : can(regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$", e))])
    error_message = "alert_emails must list at least one email address."
  }
}
