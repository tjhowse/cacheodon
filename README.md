# Cacheodon

This is an agent that relays geocaching-related events from the web to a mastodon server.

## Deployment

    git clone https://github.com/tjhowse/cacheodon
    cd cacheodon
    go build
    export GEOCACHING_CLIENT_ID=<geocaching.com username>
    export GEOCACHING_CLIENT_SECRET=<geocaching.com password>
    export MASTODON_SERVER=https://<server>
    export MASTODON_CLIENT_ID=<mastodon client id>
    export MASTODON_CLIENT_SECRET=<mastodon client secret>
    export MASTODON_USER_EMAIL=<mastodon user email address>
    export MASTODON_USER_PASSWORD=<mastodon user password>
    ./cacheodon
    <ctrl-c>

Edit config.toml to insert the coordinates and search radius you wish to monitor, E.G.

    [State]
    LastPostedFoundTime = 2023-02-28T15:47:32+10:00

    [SearchTerms]
    Latitude = -27.46794
    Longitude = 153.02809
    RadiusMeters = 16000

For Brisbane, Australia.

## Further reading

Due to the incredible bastards who designed the API at geocaching.com, I had to jump through a lot of hoops to get this working. Here's a brief overview of what I had to do.

### Background

Long ago I hid a few geocaches around Brisbane. I, and some other volunteers from the geocaching community, have maintained them for the last 15 years or so. At some point the arseholes at Groundspeak (the company that owns geocaching.com) started restricting the visibility of some geocaches, and other features, to "Premium Members". This wasn't the case when I hid my geocaches, so I developed a grudge.

### Login

When you first visit the login page (`https://www.geocaching.com/account/signin`) you get a `__RequestVerificationToken` cookie, and the POST request containing your username and password must also contain a `__RequestVerificationToken` field. However it's *not* the one you got from the cookie, it's a different one. The one it expects to see in the POST request body is actually buried in the page script. We pull it out with a regex and then use it in the body of the POST request. We provide the `__RequestVerificationToken` cookie from the original Set-Cookie header in the POST's headers because we are a good http client.

Once this is all done, we get a `gspkauth` cookie, which is the actual authentication cookie. We use this cookie for all subsequent requests.

### Searching geocaches

The search endpoint (`https://www.geocaching.com/api/proxy/web/search/v2`) accepts a bunch of URL query parameters. We provide the latitude, longitude and the radius of the search area. There is a sort parameter, but you can only use `distance` unless you are a premium member, natch. There are some `skip` and `take` parameters used for pagination. You can only get 500 records in one request, returned as gzipped JSON. The response contains a `total` field which is the total number of geocaches in the search area. We use this to calculate how many requests we need to make to get all the geocaches.

We stitch them together into one big slice, then sort by the date they were last found, then filter out the "Premium Only" geocaches (bleughh). It would've been nice to be able to filter the results by last found date, so we could stop hitting their endpoint once we'd caught up with our backlog, but noooooo.

### Retrieving logs

We want to pull out the logs of the geocaches we're interested in so we can write messages like "`person name` found `geocache`!" to mastodon. However these pricks have made it monumentally difficult to get at the logs.

First you have to obtain a special GUID for the geocache you're interested in. It's not included as a part of the search results JSON blob, you have to query it separately by hitting `https://www.geocaching.com/geocache/<geocache code>` and finding the GUID in the page script with a regex. We then send that GUID to `https://www.geocaching.com/seek/geocache_logs.aspx?guid=<GUID>` and scrape THAT page for a `userToken`. We then use THAT `userToken` to hit `https://www.geocaching.com/seek/geocache.logbook` to retrieve the logs for the geocache.

Fucking jesus christ.
