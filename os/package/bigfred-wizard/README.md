# bigfred-wizard (Buildroot package)

Fetches the prebuilt `bigfred-wizard` linux/arm64 binary from
[dcc-bigfred/bigfred-wizard](https://github.com/dcc-bigfred/bigfred-wizard)
GitHub Actions artifacts (tip or Releases) and installs it as
`/usr/sbin/bigfred-wizard`.

Party/event kiosk on the hub tablet (`:8091`). Config:
`/data/etc/bigfred/wizard/bigfred-wizard.json` (seeded on first start by the
daemon). Reverse-proxies BigFred HTTP and drives wireless-programmer for
handset commissioning.
