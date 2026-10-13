# Releasing terraform-provider-uptimewarden

Releases are built by [`.github/workflows/release.yml`](./.github/workflows/release.yml)
when a `v*` tag is pushed. GoReleaser ([`.goreleaser.yml`](./.goreleaser.yml))
builds the zip archives, writes `..._SHA256SUMS`, signs it with GPG and
attaches `..._manifest.json`. The Terraform Registry reads the GitHub release
and verifies the signature against the public key you give it.

The provider address is `registry.terraform.io/uptimewarden/uptimewarden`
(see `main.go`), so the registry namespace must be `uptimewarden`.

## One-time setup (founder)

### 1. Generate a signing key

On a trusted machine with GnuPG installed:

```bash
gpg --full-generate-key
#   kind:       (1) RSA and RSA
#   key size:   4096
#   expiry:     your choice (0 = never); renewing later is fine
#   name/email: UptimeWarden Releases <releases@uptimewarden.online> (or similar)
#   passphrase: a strong one. You will store it as a secret below.

gpg --list-secret-keys --keyid-format=long
# sec   rsa4096/<KEY_ID> ...   <- note KEY_ID
```

The registry accepts RSA and DSA keys only (not ECC), which is why the key is
RSA 4096.

### 2. Add the GitHub secrets

```bash
gpg --armor --export-secret-keys <KEY_ID> > private.asc
gh secret set GPG_PRIVATE_KEY --repo UptimeWarden/terraform-provider-uptimewarden < private.asc
gh secret set PASSPHRASE --repo UptimeWarden/terraform-provider-uptimewarden   # paste the passphrase
shred -u private.asc 2>/dev/null || rm -P private.asc
```

Or in the browser: **Settings → Secrets and variables → Actions → New
repository secret**, names `GPG_PRIVATE_KEY` (the whole armored block,
including the `BEGIN`/`END` lines) and `PASSPHRASE`.

Keep an offline backup of the private key. If it is lost, releases signed
with a new key need that new public key added to the registry as well.

### 3. Add the public key to the Terraform Registry

```bash
gpg --armor --export <KEY_ID>
```

Go to [registry.terraform.io](https://registry.terraform.io), **Sign in with
GitHub** (an account that is an admin of the `UptimeWarden` GitHub
organisation), open **Settings → Signing Keys**, choose the `uptimewarden`
namespace and paste the armored public key.

### 4. Publish the provider on the registry

Still on registry.terraform.io: **Publish → Provider**, choose the
`UptimeWarden` organisation and select the repository
`terraform-provider-uptimewarden`. The registry installs a webhook on the
repository so that every new GitHub release is picked up.

The registry only lists versions that exist as GitHub releases with the
signed checksum file, so nothing appears until step 5.

### 5. Push the first tag

```bash
git clone https://github.com/UptimeWarden/terraform-provider-uptimewarden.git
cd terraform-provider-uptimewarden
git tag v0.1.0
git push origin v0.1.0
```

Watch the **Release** workflow. When it finishes, the GitHub release holds:

- `terraform-provider-uptimewarden_0.1.0_<os>_<arch>.zip` (one per platform)
- `terraform-provider-uptimewarden_0.1.0_SHA256SUMS`
- `terraform-provider-uptimewarden_0.1.0_SHA256SUMS.sig`
- `terraform-provider-uptimewarden_0.1.0_manifest.json`

The registry ingests it within a few minutes. Check with:

```bash
cat > /tmp/uw-check.tf <<'TF'
terraform {
  required_providers {
    uptimewarden = { source = "uptimewarden/uptimewarden", version = "0.1.0" }
  }
}
TF
cd /tmp && terraform init
```

## Every later release

```bash
git tag v0.1.1   # semver; the leading v is required
git push origin v0.1.1
```

## If something fails

- **Import GPG key** step fails: a secret is missing, or `PASSPHRASE` does not
  match the key.
- Registry shows "missing signature" or "unknown key": the public key in
  **Settings → Signing Keys** is not the one whose private key is in
  `GPG_PRIVATE_KEY`.
- A version never shows up: the registry webhook only fires on releases
  created after the provider was published (step 4). For an older release,
  use **Resync** on the provider's registry page.
