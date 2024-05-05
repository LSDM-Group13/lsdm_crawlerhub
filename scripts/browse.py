import random
import webbrowser

# Read domain names from the text file into a list
with open("org_clean.txt") as f:
    domain_names = f.read().splitlines()

# Initialize a list to store domains categorized as 'yes'
yes_domains = []

browser = webbrowser.get('safari')

# Loop for 50 iterations
found = 0
while found < 125:
    # Pick a random domain from the list
    random_domain = random.choice(domain_names)

    # Open the domain in a web browser
    browser.open(random_domain)

    # Prompt for input
    user_input = input(f"Do you want to keep {random_domain}? (y/n): ")

    # Categorize the domain based on user input
    if user_input.lower() == "y":
        found += 1
        print(f"found: {found}")
        yes_domains.append(random_domain)
        # Write the 'yes' domains to a file after each input
        with open("good_domains.txt", "a") as output_file:
            output_file.write(random_domain + "\n")

print("Good domains written to 'good_domains.txt'")