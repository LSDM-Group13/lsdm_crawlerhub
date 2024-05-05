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
# query = "select * from WebPage where Data like %s"
query = f"SELECT WebPageURL, SUBSTRING(Data, GREATEST(INSTR(Data, '{search_term}') - 25, 1), LEAST(50, CHAR_LENGTH(Data) - GREATEST(INSTR(Data, '{search_term}') - 25, 1))) AS SubstringAroundMatch FROM WebPage WHERE Data LIKE %s;"
cursor.execute(query, ('%' + search_term + '%',))

print(f"Web pages with '{search_term}' in the data:")
for (webpage_url, data) in cursor:
    print(f"{webpage_url}: {data}")

cursor.close()
connection.close()