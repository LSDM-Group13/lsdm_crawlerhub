import mysql.connector

db_config = {
    "host": "localhost",
    "user": "root",
    "database": "LSDM_Group_Project"
}

domain_names = []
with open("good_domains.txt", "r") as file:
    domain_names = file.readlines()
    domain_names = [domain.strip() for domain in domain_names]

try:
    connection = mysql.connector.connect(**db_config)
    cursor = connection.cursor()

    for domain_name in domain_names:
        query = "INSERT INTO Host (DomainName) VALUES (%s)"
        cursor.execute(query, (domain_name,))

    connection.commit()
    print("Domain names inserted successfully")

except mysql.connector.Error as error:
    print("Failed to insert domain names into Host table:", error)

finally:
    if connection.is_connected():
        cursor.close()
        connection.close()