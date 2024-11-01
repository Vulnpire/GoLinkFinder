# GoLinkFinder

A minimal JS endpoint extractor

**This tool has been significantly rewritten and improved to enhance its functionality. Special thanks to 0xsha for creating this amazing tool that serves as the foundation for further development.**

# Why?

To extract endpoints in both HTML source and embedded javascript files. Useful for bug hunters, red teamers, infosec ninjas.

# Version

1.1.0

# Usage

```The -d or -f flag is required to specify a domain or an input file of URLs.
usage: goLinkFinder [-h|--help] [-d|--domain "<value>"] [-f|--file "<value>"]
                    [-o|--out "<value>"] [-s|--scope "<value>"] [-c|--complete]
Arguments:

  -h  --help         Print help information.
  -d  --domain       Input a URL to extract JavaScript links from.
  -f  --file         Specify a file containing a list of URLs (one per line).
  -o  --out          File name for output (e.g., output.txt).
  -s  --scope        Scope filtering for URLs. Use 'all' for automatic domain
                     filtering based on each URL's hostname, or specify a keyword.
  -c  --complete     Filters to only include full URLs starting with http:// or https://.
```

Basic Extraction

Use GoLinkFinder with a single domain to find all JavaScript endpoints:

`GoLinkFinder -d https://example.com`

Scoping Results

Limit results to a specific scope:

`GoLinkFinder -d https://example.com -s example`

Alternatively, use -s all to automatically filter results based on the domain names in the input URLs.

Filtering Complete URLs

To filter results to only show complete URLs (http:// or https://):

`GoLinkFinder -d https://example.com -c`

Extracting from Multiple URLs

You can also input a list of URLs from a file:

`GoLinkFinder -f urls.txt`

Output to File

Save the extracted URLs to a file:

`GoLinkFinder -d https://example.com -o output.txt`

Output :

```
 "https://api.github.com/_private/browser/stats"
 "https://api.github.com/_private/browser/errors"
 "https://github.com/github-copilot/business_signup\"
 "https://github.com/enterprise/contact?scid=&amp;utm_campaign=2023q2-site-ww-CopilotForBusiness&amp;utm_medium=referral&amp;utm_source=github\"
 "https://github.com/features/security\"
 "https://docs.github.com/get-started/learning-about-github/about-github-advanced-security#about-advanced-security-features\"
 "https://docs.github.com/enterprise-server/billing/managing-billing-for-github-advanced-security/viewing-committer-information-for-github-advanced-security\"
 "https://github.com/orgs/community/discussions/57808\"
```

you can easily pipe out its output with your other tools.

![image](https://github.com/user-attachments/assets/324a3e3f-a57d-41d0-b65a-5c4c43342bc1)


# Watch

[![asciicast](https://asciinema.org/a/HSM3Po0HC8s03XtXw3kw2UuHa.svg)](https://asciinema.org/a/HSM3Po0HC8s03XtXw3kw2UuHa)

# Requirements

Go >= 1.13

# Installation

```
go install github.com/Vulnpire/GoLinkFinder@latest
```

## Axiom Support

```
[{
        "command":"GoLinkFinder -f input -s all -c | grep -ivE '\.(png|jpe?g|gif|svg|bmp|webp|tiff?)$' | anew output",
        "ext":"txt"
}]
```

# Feature request or found an issue?

Please write a patch to fix it and then pull a request.

# References

Python implementation:
https://github.com/GerbenJavado/LinkFinder
