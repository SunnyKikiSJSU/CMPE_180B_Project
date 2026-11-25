# Team testing checklist

A manual acceptance test of the **live site** by the team, before the final
demo. The automated checks already pass ([quality-gates.md](quality-gates.md)).
This session finds what they cannot: other people using the app in their own
way, on their own phones and browsers, at the same time.

- **Site:** https://3-136-227-184.sslip.io
- **Time:** about 45-60 minutes; everyone tests at the same time (section I
  needs that).
- **Before:** ask Ahmed to start the server and database (they are stopped
  between sessions to save cost). Wait until the home page loads.

## Ground rules

- **Account:** register a new test account; do not reuse a real password.
  Use an address you control, such as `yourname+bmtest@gmail.com`. Nobody
  needs to confirm it.
- **Devices:** test on your own device and browser, and write both in the
  results table. Cover between us: Chrome, Safari, Firefox, an iPhone, and
  an Android phone.
- **Login limit:** after 10 wrong passwords from one network, login is
  blocked for a few seconds ("Too many attempts"). That is a security
  feature; wait and try again. Tell us if it ever blocks you after correct
  passwords.
- **Writing results:** for every step, write **Pass** or **Fail**. For a
  Fail, open a GitHub issue (see [Reporting a bug](#reporting-a-bug)) and
  put its number in Notes.

## Who tests what

Each person takes two sections, then everyone does I and J together.

| Tester | Sections | Device / browser |
| --- | --- | --- |
| | A + E | |
| | B + F | |
| | C + G | |
| | D + H | |
| Everyone | I, J | |

## A. Account: register, sign in, sign out

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| A1 | Open the site, click **Create an account**, register | Dashboard opens with "Welcome", four empty status shelves | | |
| A2 | Register again with the same email | "This email is already registered. Sign in instead." | | |
| A3 | Register with a password under 8 characters, or a blank name | A message under the field; nothing is created | | |
| A4 | Sign out, then sign in with a wrong password | "Email or password is incorrect." | | |
| A5 | Sign in with the right password | Dashboard | | |
| A6 | Signed out, open `/shelves` directly | Sent to the sign-in page; after signing in, back on Shelves | | |
| A7 | Close the browser, reopen the site | Still signed in (the session lasts about an hour) | | |

## B. Catalog and Open Library

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| B1 | **Catalog** → Open Library tab, search a book you know | Results with covers within a few seconds | | |
| B2 | Add one result to a shelf (**Want to read**) | It is imported; the book page shows it on Want to read | | |
| B3 | Import the same book again (from your account or another teammate's) | No duplicate: the same book opens | | |
| B4 | Library tab: search by title, then filter by author and by category (**More filters**) | Matching books only; the counts look right | | |
| B5 | Try each sort (Best match, Newest first, Highest rated, Most popular) | The order changes accordingly | | |
| B6 | Page through results (next/previous) | No repeated or missing books between pages | | |
| B7 | Signed out, open **Explore** | Public catalog works without an account | | |

## C. Book page, authors, categories

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| C1 | Open a book | Title, cover, authors, categories, description, rating summary | | |
| C2 | Click an author | Author page lists their books | | |
| C3 | Click a category | Category page lists its books | | |
| C4 | Open `/books/not-a-real-id` | A friendly "not found" page, not a crash | | |
| C5 | Note anything that looks wrong in descriptions (for example raw `[text](link)` markup) | Write it down; known cosmetic issue on some Open Library books | | |

## D. Shelves

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| D1 | **Shelves** → create a custom shelf "Favorites" | It appears under Custom shelves | | |
| D2 | Create another shelf with the same name | Rejected with a clear message | | |
| D3 | Rename the custom shelf, then try to rename a status shelf | Custom: renamed. Status shelf: not allowed | | |
| D4 | Move a book Want to read → Currently reading | It leaves Want to read; counts update | | |
| D5 | Add the same book to "Favorites" (**Also add to**) | It is on both Currently reading and Favorites | | |
| D6 | Remove the book from Currently reading (confirm) | It leaves the status shelf and its progress is cleared; still on Favorites | | |
| D7 | Delete the custom shelf | Shelf gone; the books still exist in the catalog | | |

## E. Reading progress

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| E1 | On a book with a page count, set page 50 | Status Reading; percent shown; book on Currently reading | | |
| E2 | Enter a page larger than the book | Rejected with a message; nothing changes | | |
| E3 | Enter the last page | Status becomes **Finished** automatically; book moves to Finished; history says "Finished the book" | | |
| E4 | Open the same book in **two tabs**, save a page in tab 1, then a different page in tab 2 | Tab 2 says the progress changed elsewhere and offers to reload; nothing is lost | | |
| E5 | **Progress** page: filter by status | Only books with that status | | |

## F. Reviews and ratings

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| F1 | Rate a book 4 stars | Summary updates (average and count) | | |
| F2 | Change the rating to 2, then remove it | Summary updates each time | | |
| F3 | Write a review with a rating | Appears on the book page; review count +1 | | |
| F4 | Try to post an empty review | Rejected with a message | | |
| F5 | Edit, then delete your review | Edited text shows; after delete the count goes back | | |
| F6 | **Reviews** page | Your reviews with book titles | | |
| F7 | Open a teammate's review | No edit or delete buttons on someone else's review | | |

## G. Reports and dashboard

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| G1 | **Reports**: open every tab (My reading, Popular, Top rated, Author ranking, Category ranking, Review activity, SQL queries) | Each loads with data, no error | | |
| G2 | Change **Window** on Popular and Review activity, and **Minimum** on Top rated and Author ranking | Results change; the URL keeps the choice after a reload | | |
| G3 | My reading → **Recommended for you** | Suggests books you have not shelved yet | | |
| G4 | Dashboard after sections D-F | Counts, pages read, and history reflect what you did | | |

## H. Settings, layout, and errors

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| H1 | **Settings**: change display name and bio | Saved; the new name shows on your reviews | | |
| H2 | Change the password with a wrong current password | Rejected under "Current password" | | |
| H3 | Change it correctly, sign out, sign in with the new one | Works | | |
| H4 | On a phone: open the menu (drawer), go through every page | Nothing overlaps or is cut off; buttons are tappable | | |
| H5 | Rotate the phone, and try a narrow desktop window | Layout adapts | | |
| H6 | Open `/this-page-does-not-exist` | Friendly 404 page | | |

## I. Everyone at once (about 5 minutes)

Pick one book everyone can find (for example search "Earthsea").

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| I1 | On a count of three, everyone rates the book | Rating count on the book = number of people; average = average of what you chose | | |
| I2 | Everyone writes a short review at the same time | Every review appears once; review count matches | | |
| I3 | Everyone reloads Reports → Popular and Top rated | Pages load quickly; no errors | | |
| I4 | Everyone signs out and back in within a minute | No one is blocked by the login limit | | |

## J. Delete the test accounts (last)

| # | Step | Expected | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| J1 | **Settings** → Delete account (confirm) | Signed out; signing in with that account fails | | |
| J2 | Open the shared book from section I | Your rating and review are gone; the counts dropped accordingly | | |

## Reporting a bug

Open an issue at https://github.com/SunnyKikiSJSU/CMPE_180B_Project/issues
with:

1. **Title:** short and specific ("Progress page: filter shows finished books under Reading").
2. **Steps:** exactly what you did, starting from a page URL.
3. **Expected** vs **what happened**.
4. **Device and browser** (for example iPhone 15, Safari 18).
5. **Screenshot**, and the **Reference** code if an error box shows one
   (it matches the server log line).

Never paste a password into an issue.

## Results

| Date | Testers | Devices / browsers | Steps passed | Steps failed | Issues opened |
| --- | --- | --- | --- | --- | --- |
| | | | | | |

After the session, ask Ahmed to stop the server and database.
