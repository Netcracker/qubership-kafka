# Kafka Helm Chart

The documentation for Kafka Service installation using Helm can be found in [Installation Using Helm](/docs/public/installation.md).

## Backup Daemon S3 aliases

Use these values to configure named backup-storage aliases for backup/restore flows:

- `backupDaemon.s3Aliases` - list of S3 alias definitions. If empty, no aliases secret is rendered.

Example:

```yaml
backupDaemon:
  s3Aliases:
    - name: default
      spec:
        default: true
        storageBucket: backup-restore-bucket
        storageProvider: aws
        storageRegion: us-east-1
        storageServerUrl: "https://s3.example.com"
        storageUsername: name
        storageSecret: storage-location
      secretContent:
        storagePassword: pass
```
