# Azurite - Azure Storage API Emulator

## How to Run
```bash
lab up azurite
```

## How to Use
- **Endpoints** (direct IP:Port):
    - Blob: `192.168.1.8:10000`
    - Queue: `192.168.1.8:10001`
    - Table: `192.168.1.8:10002`

## Credentials
- **Default Account**: `devstoreaccount1`
- **Default Key**: `Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==`
- **Custom Accounts**: Defined in `.env` via `AZURITE_ACCOUNTS`.

### Default Connection String
```text
DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==;BlobEndpoint=http://192.168.1.8:10000/;QueueEndpoint=http://192.168.1.8:10001/;TableEndpoint=http://192.168.1.8:10002/;
```

## Networking
- **Direct Access**: Accessible via `192.168.1.8:10000-10002`.
- **Production**: Not exposed via the proxy. Access via local network only.
