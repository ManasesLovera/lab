# Getting docker permissions locally

- Adding User to Docker group

```bash
sudo usermod -aG docker $USER
```

- Activate the changes

```bash
newgrp docker
```
