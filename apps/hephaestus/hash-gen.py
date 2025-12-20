import bcrypt
import sys

password = b"HephaestusPreserver"
hashed = bcrypt.hashpw(password, bcrypt.gensalt())
print(hashed.decode('utf-8'))
