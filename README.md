# ratt
Recon All The Things (R.A.T.T.) is a first step recon tool for seeding resources to be fed into other recon tools. RATT performs the following steps for each domain fed to it.

1. Creates directory named after the hostname of the target
2. Fetches contents of URL and save the HTML to contents.html
3. Creates a metadata file containing the URL fetched, the headers returned in the response, and the title of the page
4. Finds all relative link tags (<a>) and save them to relativePaths.txt
5. Finds all absolute link tags (<a>) and save them to absolutePaths.txt
6. Finds all inline Javascript tags and saves their contents to a generated file name at /js/<generated name>
7. Finds all imported script tags (<script src="<some path>/<some file.js>"), fetches the resource, and saves it to /js/<some file>.js

# Further Work
- Crawl JS files for relative/absolute paths
- Recusively walking steps 4 & 5 on paths found through HTML/JS discovery to find more data
- Things I haven't though of yet ;)
