************************
shform: Shared Form Demo
************************

This demonstration sketches out a design for implementing a web form
that allows concurrent distributed users to observe one another's
edits in real time.

**Table of Contents**

.. contents::
   :local:
   :depth: 1
   :backlinks: none

======================
Architectural Overview
======================

From the user inward, these are the main components of this demo:

* *Twitter Bootstrap* --- For easy mobile support and pleasing
  presentation, `bootstrap <http://getbootstrap.com/>`_ is used for
  styling the front end.
* *Knockout JS* --- We want user actions like text edits to
  immediately trigger updates in the browsers of all the other users.
  We also want each user's form elements to immediately reflect edits
  made by remote users.  The first step is to bind the form data to
  the javascript in the browser, so we use `Knockout JS
  <http://knockoutjs.com/>`_, a JavaScript library that focuses on
  data binding.

  Knockout provides a rate-limiting feature that allows natural
  throttling/batching of events.
* *WebSocket* --- The JavaScript running in the browser communicates
  asynchronously with a back-end server via full-duplex reliable
  channel using the standard HTML 5 WebSocket feature.
* *Gorilla* --- The web server, implemented in `Go
  <http://golang.org/>`_, uses the `Gorilla toolkit
  <http://www.gorillatoolkit.org/>`_, which provides high-level
  features like session support and websocket support.

================
Missing Features
================

Much is missing from this simple proof of concept, including ...

* User identities --- A user should have a way to self identify, so
  that each user can tell who is making what changes.

  These user identities should be associated with the updates made to
  the data, and this association should persist in the back end.  This
  persistent data has the potential to allow users to see who made
  what changes, increasing their confidence in the software.
* Oauth or other --- The users could be authenticated using a trusted
  technology like `OAuth <http://en.wikipedia.org/wiki/OAuth>`_.
* Visual Remote-Edit Cues --- When a remote user makes an edit, the
  local user needs a way to quickly know that something has changed
  and how.

  One idea would be to outline the changing field in a highlight color
  and append an italicized (*"Eugene is editing ..."*) text string to
  the field's label.
* Validate CSRF --- To foil cross-site request forgery (CSRF), the
  CSRF token that is already included in the HTML would need to be
  sent by the client to the server and validated.

  I tried out Gorilla's session support and used core Go team member
  Andrew Gerrit's `XSRF module
  <http://godoc.org/code.google.com/p/xsrftoken>`_ to make sure this
  feature wouldn't be too hard to add.
* Persistence of data --- The server would store data in a database,
  probably in two kinds:

  * *Current* data is being edited by the users, and
  * *Saved* data has been explicitly committed by a user via a "save"
    button.  This distinction gives the users a way to indicate that
    the data has reached some significant stopping point.  These saved
    versions of the form effectively become snapshots if they're
    retained indefinitely.

* Flakey networking tolerance --- Especially with mobile, the network
  could come and go.  The client needs to be able to re-establish a
  WebSocket connection after losing connectivity.

  Depending on the situation, full offline support might be important
  to provide, allowing users with no network connectivity the option
  of continuing to make edits and providing a way to merge offline
  changes when connectivity resumes.
* Fall-back mechanism --- If supporting users with old or lame
  browsers is desireable, long-polling AJAX can be used instead of
  WebSockets.  There are libraries like `Sock JS
  <https://github.com/sockjs>`_ and `Socket.IO <http://socket.io/>`_
  for transparently providing fall back behavior.  The server would
  need something to support the client when it falls back, e.g., a
  `RESTful interface
  <http://en.wikipedia.org/wiki/Representational_state_transfer>`_
  using `JSON <http://www.json.org/>`_.

==============
Data Conflicts
==============

When users make edits concurrently, it is possible for their edits to
interfere, such that one user "wins", while the other user's edits are
lost.

The current implementation minimizes the impact of such potential data
loss by communicating edits quickly, so that the amount of change is
small for interactive (human typing) form use.

To handle larger edits (e.g., copy-and-paste) or eliminate the
potential for any data loss, concurrency could be actively limited by
the application.  The front end would handle gain-focus events in form
elements by requesting exclusive write access to the element's state.
Only after the server confirmed the exclusive access would the front
end allow modification of the form element's state.  This
implementation is more complex and more likely to inconvenience the
user.  The current implementation is a practical compromise that is
expected to feel natural to the users.
