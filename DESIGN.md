# Design Document

In this document I will go through and describe my thought process and design decisions for this application.

- [Requirements](#requirements)
- [Questions](#questions)
- [Design](#design)
  - [Interface](#interface)
- [Decisions](#decisions) 

## Requirements

- The application:
  - MUST fetch content from a given URL.
  - MUST recursively parse links within a HTML document.
  - MUST fetch content from those parsed links, 
    - when the domain IS EQUAL to the start domain.
    - when the domain IS NOT an external domain (e.g. facebook.com or community.monzo.com). 
  - MUST print each URL visited into standard out.


## Open Questions

### What is exactly meant by "visited"?

If for example a page returns a non success response do we see that as being "visited"? For the moment I will keep it simple and I will say any page
that we get a response from we have visited, even if it is a "4xx" or a "5xx". However I will keep it in mind that we might want to change this
when implementing.

### Should a sub-domain be considered an external URL?

In the requirements I think we have a contradiction  community.monzo.com is a subdomain of monzo.com, so it should be called, 
it's not an external domain but in the requirements we call it out as an external domain? For the moment I will leave this out, 
but I will keep it in mind if we do want to implement it in the future.

### How should we handle pages which redirect?

Should we follow or should we not follow those pages which redirect, for the moment to keep it simple I will opt to not follow redirects but if we do
in the future I would suggest that is its own piece of work which needs to be defined, because there will be decisions to make there.


### Should we respect robots.txt

I think we should, but might not have time so we'll see.

## Design

### Architecture

#### High level

At the core of this application will be a queue which contains a list of valid URLs to visit, the application will then take a url from that list and then request the contents from that URL, when that page returns a response we then print and any links that have not been visited before (and are from the same domain as the start url) will then be appended to that queue.


#### Specifics

Although the requirements are not clear around the performance, it is safe to assume we should write the application in a scalable way so that we can easily increase the performance. In order to have a scalable and maintainable application, lets split the components of this application out into distinct parts, so that tasks can be performed concurrently. With this structure we should be able to scale the number of "requesters", "parsers", "link validators".


