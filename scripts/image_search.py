from cassandra.cluster import Cluster
from PIL import Image
from io import BytesIO
import mysql.connector
import sys
import warnings

connection = mysql.connector.connect(
    host="localhost",
    user="root",
    database="LSDM_Group_Project"
)

cluster = Cluster(['localhost'])
session = cluster.connect('lsdm_images')

if not connection.is_connected():
    print("Failed to connect to the database.")
    sys.exit(1)

if len(sys.argv) < 2:
    print("Enter a search term")

show_one = False
if len(sys.argv) > 2:
    show_one = True

search_term = sys.argv[1]

cursor = connection.cursor()
query = f"SELECT WebPageURL, WebPageID, SUBSTRING(Data, GREATEST(INSTR(Data, '{search_term}') - 25, 1), LEAST(50, CHAR_LENGTH(Data) - GREATEST(INSTR(Data, '{search_term}') - 25, 1))) AS SubstringAroundMatch FROM WebPage WHERE Data LIKE %s;"
cursor.execute(query, ('%' + search_term + '%',))

print(f"Web pages with '{search_term}' in the data:")
for (webpage_url, webpage_id, data) in cursor:
    print(f"{webpage_url}: {data}")
    query = "SELECT image_data FROM images WHERE webpage_id = %s ALLOW FILTERING"
    result = session.execute(query, (int(webpage_id),))

    if show_one:
        for i in range(2):
            with warnings.catch_warnings():
                warnings.simplefilter("ignore", category=DeprecationWarning)
                image_data = result[i].image_data
            image = Image.open(BytesIO(image_data))
            image.show()
        break

    for row in result:
        image_data = row.image_data
        image = Image.open(BytesIO(image_data))
        image.show()


session.shutdown()
cursor.reset()
cursor.close()
connection.close()




