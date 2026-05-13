config {
  format = "compact"
}

plugin "terraform" {
  enabled = true
  preset  = "recommended"
}

plugin "aws" {
  enabled = true
  version = "0.32.0"
  source  = "github.com/terraform-linters/tflint-ruleset-aws"
}

# Modules inherit provider/version constraints from root envs in this PoC repo.
rule "terraform_required_version" {
  enabled = false
}

rule "terraform_required_providers" {
  enabled = false
}

# Shared module interfaces intentionally keep project/env for future tags even when a
# specific module currently consumes only the merged tags map.
rule "terraform_unused_declarations" {
  enabled = false
}
