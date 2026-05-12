# Azurite - Azure Storage API Emulator

## How to Run
```bash
lab up azurite
```

## How to Use
- **Endpoints**:
    - Blob: `azurite.rpi.local:10000`
    - Queue: `azurite.rpi.local:10001`
    - Table: `azurite.rpi.local:10002`

## Credentials
- **Default Account**: `devstoreaccount1`
- **Default Key**: `Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==`
- **Custom Accounts**: Defined in `.env` via `AZURITE_ACCOUNTS`.

### Default Connection String
```text
DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==;BlobEndpoint=http://azurite.rpi.local:10000/;QueueEndpoint=http://azurite.rpi.local:10001/;TableEndpoint=http://azurite.rpi.local:10002/;
```

## Networking
- **Local Network**: Accessible via `azurite.rpi.local`.
- **Production**: Remote access via Traefik can be enabled by adding the `Host('azurite.mlovera.dev')` label.
