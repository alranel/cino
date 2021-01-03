# cino-server

cino-server is the orchestrator for a cino setup. It exposes:

* an HTTP endpoint for receiving GitHub event notifications;
* a PostgreSQL server for job management and pub/sub communication with runners.

It also reports job status to GitHub whenever updates from runners are available.

## Installing cino-runner

A docker-compose.yml file is provided for convenience. Setting it up is very easy:

1. Configure the app on GitHub following their [instructions](https://docs.github.com/en/free-pro-team@latest/developers/apps/creating-a-github-app). There's no need to configure OAuth, but you'll need to generate and download a private key. As webhook URL you'll enter `http://<hostname>:8080/github-hook`.

    > The supplied docker-compose.yml does not include TLS, but adding a reverse proxy like traefik is trivial.

2. Clone the cino repo:

    ```
    mkdir /srv/cino
    git clone https://github.com/alranel/cino.git /srv/cino
    cd /srv/cino
    ```

3. Generate a TLS private key for PostgreSQL:

    ```
    openssl req -nodes -new -x509 -keyout server.key -out server.cert -subj /CN=localhost
    chmod 600 server.key
    ```

4. Set a password for PostgreSQL:

    ```
    cp .env.example .env
    vi .env
    ```

    This file contains the credentials used by all runners to access the database. All of them will access the whole database and no untrusted runners are supposed to be involved, so there's no strict need for distinct credentials.

5. Copy and populate the configuration file:

    ```
    cp config.yml.example config.yml
    vi config.yml
    ```

    * **github.app_id**: the ID of the GitHub app
    * **github.secret**: the secret used to validate the GitHub event notifications
    * **github.private_key_file**: the path to the private key generated by GitHub to [authenticate to their API](https://docs.github.com/en/free-pro-team@latest/developers/apps/authenticating-with-github-apps)
    * **runners**: the list of cino-runner instances that are supposed to be always connected to this server. Their IDs can be freely assigned, as long as they are unique strings. Make sure no inactive runners are listed, otherwise jobs may stall waiting for them.
    * **architectures**: the list of architectures supported by our CI pool. This is used to generate the CI jobs for libraries.

6. Start the server:

    ```
    docker-compose build
    docker-compose up -d
    ```

You can now proceed with the configuration of your [cino-runner instances](../cino-runner).