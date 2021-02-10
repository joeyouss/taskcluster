#![allow(unused_imports)]
#![cfg_attr(rustfmt, rustfmt_skip)]
/* THIS FILE IS AUTOMATICALLY GENERATED. DO NOT EDIT */
use crate::{Client, ClientBuilder, Credentials, Retry};
use anyhow::Error;
use serde_json::Value;
use std::time::Duration;
use crate::util::urlencode;

/// Notification Service
///
/// The notification service listens for tasks with associated notifications
/// and handles requests to send emails and post pulse messages.
pub struct Notify (Client);

#[allow(non_snake_case)]
impl Notify {
    /// Create a new undefined instance, based on the given client.
    pub fn new<CB: Into<ClientBuilder>>(client_builder: CB) -> Result<Self, Error> {
        Ok(Self(client_builder
            .into()
            .path_prefix("api/notify/v1/")
            .build()?))
    }

    /// Ping Server
    /// 
    /// Respond without doing anything.
    /// This endpoint is used to check that the service is up.
    pub async fn ping(&self) -> Result<(), Error> {
        let method = "GET";
        let (path, query) = Self::ping_details();
        let body = None;
        let resp = self.0.request(method, path, query, body).await?;
        resp.bytes().await?;
        Ok(())
    }

    /// Generate an unsigned URL for the ping endpoint
    pub fn ping_url(&self) -> Result<String, Error> {
        let (path, query) = Self::ping_details();
        self.0.make_url(path, query)
    }

    /// Generate a signed URL for the ping endpoint
    pub fn ping_signed_url(&self, ttl: Duration) -> Result<String, Error> {
        let (path, query) = Self::ping_details();
        self.0.make_signed_url(path, query, ttl)
    }

    /// Determine the HTTP request details for ping
    fn ping_details<'a>() -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "ping";
        let query = None;

        (path, query)
    }

    /// Send an Email
    /// 
    /// Send an email to `address`. The content is markdown and will be rendered
    /// to HTML, but both the HTML and raw markdown text will be sent in the
    /// email. If a link is included, it will be rendered to a nice button in the
    /// HTML version of the email
    pub async fn email(&self, payload: &Value) -> Result<(), Error> {
        let method = "POST";
        let (path, query) = Self::email_details();
        let body = Some(payload);
        let resp = self.0.request(method, path, query, body).await?;
        resp.bytes().await?;
        Ok(())
    }

    /// Determine the HTTP request details for email
    fn email_details<'a>() -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "email";
        let query = None;

        (path, query)
    }

    /// Publish a Pulse Message
    /// 
    /// Publish a message on pulse with the given `routingKey`.
    pub async fn pulse(&self, payload: &Value) -> Result<(), Error> {
        let method = "POST";
        let (path, query) = Self::pulse_details();
        let body = Some(payload);
        let resp = self.0.request(method, path, query, body).await?;
        resp.bytes().await?;
        Ok(())
    }

    /// Determine the HTTP request details for pulse
    fn pulse_details<'a>() -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "pulse";
        let query = None;

        (path, query)
    }

    /// Post Matrix Message
    /// 
    /// Post a message to a room in Matrix. Optionally includes formatted message.
    /// 
    /// The `roomId` in the scopes is a fully formed `roomId` with leading `!` such
    /// as `!foo:bar.com`.
    /// 
    /// Note that the matrix client used by taskcluster must be invited to a room before
    /// it can post there!
    pub async fn matrix(&self, payload: &Value) -> Result<(), Error> {
        let method = "POST";
        let (path, query) = Self::matrix_details();
        let body = Some(payload);
        let resp = self.0.request(method, path, query, body).await?;
        resp.bytes().await?;
        Ok(())
    }

    /// Determine the HTTP request details for matrix
    fn matrix_details<'a>() -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "matrix";
        let query = None;

        (path, query)
    }

    /// Post Slack Message
    /// 
    /// Post a message to a Slack channel.
    /// 
    /// The `channelId` in the scopes is a Slack channel ID, starting with a capital C.
    /// 
    /// The Slack app can post into public channels by default but will need to be added
    /// to private channels before it can post messages there.
    pub async fn slack(&self, payload: &Value) -> Result<(), Error> {
        let method = "POST";
        let (path, query) = Self::slack_details();
        let body = Some(payload);
        let resp = self.0.request(method, path, query, body).await?;
        resp.bytes().await?;
        Ok(())
    }

    /// Determine the HTTP request details for slack
    fn slack_details<'a>() -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "slack";
        let query = None;

        (path, query)
    }

    /// Denylist Given Address
    /// 
    /// Add the given address to the notification denylist. Addresses in the denylist will be ignored
    /// by the notification service.
    pub async fn addDenylistAddress(&self, payload: &Value) -> Result<(), Error> {
        let method = "POST";
        let (path, query) = Self::addDenylistAddress_details();
        let body = Some(payload);
        let resp = self.0.request(method, path, query, body).await?;
        resp.bytes().await?;
        Ok(())
    }

    /// Determine the HTTP request details for addDenylistAddress
    fn addDenylistAddress_details<'a>() -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "denylist/add";
        let query = None;

        (path, query)
    }

    /// Delete Denylisted Address
    /// 
    /// Delete the specified address from the notification denylist.
    pub async fn deleteDenylistAddress(&self, payload: &Value) -> Result<(), Error> {
        let method = "DELETE";
        let (path, query) = Self::deleteDenylistAddress_details();
        let body = Some(payload);
        let resp = self.0.request(method, path, query, body).await?;
        resp.bytes().await?;
        Ok(())
    }

    /// Determine the HTTP request details for deleteDenylistAddress
    fn deleteDenylistAddress_details<'a>() -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "denylist/delete";
        let query = None;

        (path, query)
    }

    /// List Denylisted Notifications
    /// 
    /// Lists all the denylisted addresses.
    /// 
    /// By default this end-point will try to return up to 1000 addresses in one
    /// request. But it **may return less**, even if more tasks are available.
    /// It may also return a `continuationToken` even though there are no more
    /// results. However, you can only be sure to have seen all results if you
    /// keep calling `list` with the last `continuationToken` until you
    /// get a result without a `continuationToken`.
    /// 
    /// If you are not interested in listing all the members at once, you may
    /// use the query-string option `limit` to return fewer.
    pub async fn listDenylist(&self, continuationToken: Option<&str>, limit: Option<&str>) -> Result<Value, Error> {
        let method = "GET";
        let (path, query) = Self::listDenylist_details(continuationToken, limit);
        let body = None;
        let resp = self.0.request(method, path, query, body).await?;
        Ok(resp.json().await?)
    }

    /// Generate an unsigned URL for the listDenylist endpoint
    pub fn listDenylist_url(&self, continuationToken: Option<&str>, limit: Option<&str>) -> Result<String, Error> {
        let (path, query) = Self::listDenylist_details(continuationToken, limit);
        self.0.make_url(path, query)
    }

    /// Generate a signed URL for the listDenylist endpoint
    pub fn listDenylist_signed_url(&self, continuationToken: Option<&str>, limit: Option<&str>, ttl: Duration) -> Result<String, Error> {
        let (path, query) = Self::listDenylist_details(continuationToken, limit);
        self.0.make_signed_url(path, query, ttl)
    }

    /// Determine the HTTP request details for listDenylist
    fn listDenylist_details<'a>(continuationToken: Option<&'a str>, limit: Option<&'a str>) -> (&'static str, Option<Vec<(&'static str, &'a str)>>) {
        let path = "denylist/list";
        let mut query = None;
        if let Some(q) = continuationToken {
            query.get_or_insert_with(Vec::new).push(("continuationToken", q));
        }
        if let Some(q) = limit {
            query.get_or_insert_with(Vec::new).push(("limit", q));
        }

        (path, query)
    }
}