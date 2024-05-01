import mysql.connector
import sys

connection = mysql.connector.connect(
    host="localhost",
    user="root",
    database="LSDM_Group_Project"
)

if not connection.is_connected():
    print("Failed to connect to the database.")
    sys.exit(1)

search_term = sys.argv[1]

cursor = connection.cursor()
query = "select * from WebPage where Data like %s"
cursor.execute(query, ('%' + search_term + '%',))

print(f"Web pages with '{search_term}' in the data:")
for (webpage_id, host_id, webpage_url, data) in cursor:
    print(f"{webpage_url}: {data}")

cursor.close()
connection.close()