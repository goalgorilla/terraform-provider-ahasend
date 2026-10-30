# Provider-account credential
terraform import ahasend_smtp_credential.example 00000000-0000-0000-0000-000000000030

# Sub-account credential: ImportState does not store api_key. Import while the
# resource uses a child provider alias so the first Read authenticates as the
# child, then switch to resource-level api_key (see sub-account-child-domain guide).
terraform import ahasend_smtp_credential.example 00000000-0000-0000-0000-0000000000aa/00000000-0000-0000-0000-000000000030
