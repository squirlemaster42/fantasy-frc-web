use axum::{routing::get, Router};
use std::net::SocketAddr;
use tower_http::cors::{Any, CorsLayer};

extern crate dotenv;
use dotenv::dotenv;

mod types;
mod tba_client;

#[tokio::main]
async fn main() {
    dotenv().ok();

    println!("{}", tba_client::make_tba_match_list_request("2023cur").await.into_iter().nth(0).unwrap().key);

    let cors = CorsLayer::new().allow_origin(Any);

    let app = Router::new()
        .route("/", get(root))
        .layer(cors);

    let addr = SocketAddr::from(([127, 0, 0, 1], 3000));
    println!("listening on {}", addr);

    axum::Server::bind(&addr)
        .serve(app.into_make_service())
        .await
        .unwrap();
}

async fn root() -> &'static str {
    "Hello, World!"
}
