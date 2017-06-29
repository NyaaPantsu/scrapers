import cfscrape
import sys

scraper = cfscrape.create_scraper()
url = sys.argv[1]
print scraper.get(url).content