# Quant Provider

The Quant provider allows you to manage resources in your Quant CDN environment.

## Example Usage

```hcl
provider "quant" {
  bearer = "fxj1eivEXhKdIEVGuKkfrcfv4WeEQ8uqqNqEeIEy4zEb6hlz8Tj1SdRxdc9x"
  organization = "quant"
}
```

# Required configuration

- `bearer` - The bearer token for authentication.
- `organization` - The organization name.
