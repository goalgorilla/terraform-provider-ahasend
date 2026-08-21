# Why `dns_valid` does not fail apply

Sending domains are often created before DNS is published. AhaSend returns the records you must create (`dns_records`) and whether validation currently passes (`dns_valid`).

Terraform apply succeeds even when `dns_valid` is false so you can:

1. Create the domain and capture `dns_records`
2. Publish CNAMEs/TXT/MX at your DNS provider
3. Re-run apply/refresh with `check_dns = true` until `dns_valid` becomes true

Failing the apply on incomplete DNS would block that workflow and couple AhaSend lifecycle to external DNS propagation.
