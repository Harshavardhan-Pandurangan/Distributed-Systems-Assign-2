import os
import json
import subprocess

BASE_DIR = os.getcwd()
NO_OF_CLIENTS = 10
NO_OF_BANKS = 5


def verify_certificate(cert_path, key_path, ca_cert_path):
    """Verifies certificate validity and key matching."""
    try:
        # Check certificate dates
        subprocess.run(['openssl', 'x509', '-in', cert_path,
                       '-noout', '-dates'], check=True)

        # Verify certificate issuance
        subprocess.run(['openssl', 'x509', '-in', cert_path,
                       '-noout', '-subject', '-issuer'], check=True)

        # Verify certificate chain
        subprocess.run(['openssl', 'verify', '-CAfile',
                       ca_cert_path, cert_path], check=True)

        # Verify key matching
        cert_modulus = subprocess.run(['openssl', 'x509', '-noout', '-modulus', '-in', cert_path],
                                      capture_output=True, text=True, check=True).stdout.split('=')[1].strip()
        key_modulus = subprocess.run(['openssl', 'rsa', '-noout', '-modulus', '-in', key_path],
                                     capture_output=True, text=True, check=True).stdout.split('=')[1].strip()

        if cert_modulus != key_modulus:
            print(
                f"ERROR: Private key does not match certificate: {cert_path}")
            return False

        return True

    except subprocess.CalledProcessError as e:
        print(f"Verification failed for {cert_path}: {e}")
        return False


if __name__ == '__main__':
    # Create CA certificate files in BASE_DIR
    os.system(f'openssl genrsa -out {BASE_DIR}/ca.key 4096')
    os.system(
        f'openssl req -x509 -new -nodes -key {BASE_DIR}/ca.key -sha256 -days 365 -out {BASE_DIR}/ca.crt -subj "/CN=StrifeCA"')

    ca_cert_path = f'{BASE_DIR}/ca.crt'
    ca_key_path = f'{BASE_DIR}/ca.key'

    # verify CA
    if not verify_certificate(ca_cert_path, ca_key_path, ca_cert_path):
        print("CA certificate verification failed.")
        exit(1)
    else:
        print("CA certificate verified.")

    os.chdir('cmd/gateway')
    # Create server certificate
    os.system('openssl genrsa -out server.key 2048')
    os.system(
        'openssl req -new -key server.key -out server.csr -subj "/CN=StrifeGateway"')

    # create v3.ext file for SAN
    with open('v3.ext', 'w') as f:
        f.write("""
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
""")

    os.system(
        f'openssl x509 -req -in server.csr -CA {BASE_DIR}/ca.crt -CAkey {BASE_DIR}/ca.key -CAcreateserial -out server.crt -days 365 -sha256 -extfile v3.ext')

    server_cert_path = 'server.crt'
    server_key_path = 'server.key'

    if not verify_certificate(server_cert_path, server_key_path, f'{BASE_DIR}/ca.crt'):
        print("Server certificate verification failed.")
        exit(1)
    else:
        print("Server certificate verified.")

    os.chdir('../client/certs')
    # create admin certificate
    os.system('openssl genrsa -out client0.key 2048')
    os.system(
        'openssl req -new -key client0.key -out client0.csr -subj "/CN=StrifeAdmin"')
    os.system(
        f'openssl x509 -req -in client0.csr -CA {BASE_DIR}/ca.crt -CAkey {BASE_DIR}/ca.key -CAcreateserial -out client0.crt -days 365 -sha256')

    admin_cert_path = 'client0.crt'
    admin_key_path = 'client0.key'

    if not verify_certificate(admin_cert_path, admin_key_path, f'{BASE_DIR}/ca.crt'):
        print("Admin certificate verification failed.")
        exit(1)
    else:
        print("Admin certificate verified.")

    for c_no in range(1, NO_OF_CLIENTS + 1):
        # Create client certificates
        os.system(f'openssl genrsa -out client{c_no}.key 2048')
        os.system(
            f'openssl req -new -key client{c_no}.key -out client{c_no}.csr -subj "/CN=StrifeClient{c_no}"')
        os.system(
            f'openssl x509 -req -in client{c_no}.csr -CA {BASE_DIR}/ca.crt -CAkey {BASE_DIR}/ca.key -CAcreateserial -out client{c_no}.crt -days 365 -sha256')

        client_cert_path = f'client{c_no}.crt'
        client_key_path = f'client{c_no}.key'

        if not verify_certificate(client_cert_path, client_key_path, f'{BASE_DIR}/ca.crt'):
            print(f"Client {c_no} certificate verification failed.")
            exit(1)
        else:
            print(f"Client {c_no} certificate verified.")

    os.chdir(BASE_DIR)

    # create a json file for each client
    for c_no in range(NO_OF_CLIENTS):
        client_data = {
            'client': {
                'name': f'StrifeClient{c_no}',
                'transactions': [],
            },
        }

        # write the client data to a json file
        with open(f'cmd/client/databases/client{c_no}.json', 'w') as f:
            json.dump(client_data, f, indent=4)

    # Create server and bank database files
    server_data = {
        'server': {
            'name': 'StrifeGateway',
            'entries': [],
            'transactions': [],
        },
    }

    # write the server data to a json file
    with open('cmd/gateway/server.json', 'w') as f:
        json.dump(server_data, f, indent=4)

    # create a json file for each bank
    for b_no in range(NO_OF_BANKS):
        bank_data = {
            'bank': {
                'name': f'StrifeBank{b_no}',
                'entries': [],
                'transactions': [],
            },
        }

        # write the bank data to a json file
        with open(f'cmd/bank/databases/bank{b_no}.json', 'w') as f:
            json.dump(bank_data, f, indent=4)
