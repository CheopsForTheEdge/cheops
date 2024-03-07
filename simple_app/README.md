# Simple App

Simple App is a simple CRUD application making resources available. The goal is to be able to act on these resources through multiple means. Each resource is of a defined type, each type has specific operations.

All operations are done with a POST at /{id}?type=XXX&operation=XXX&value=XXX
All resources can be fetched with a GET at /{id}?type=XXX

## Data types

### Counter

A counter is an integer that can go to any positive or negative values. In the query parameters, it means type=counter. The available operations are:

- insert: set the counter to a specific value S (operation=insert&value=S)
- add: add the (possibly negative) value V to the counter (operation=add&value=V)