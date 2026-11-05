use serde::Serialize;

#[derive(Serialize)]
struct Sobre {
    nome: String,
    versao: String,
}

#[tauri::command]
fn sobre(app: tauri::AppHandle) -> Sobre {
    let info = app.package_info();
    Sobre {
        nome: info.name.clone(),
        versao: info.version.to_string(),
    }
}

pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![sobre])
        .run(tauri::generate_context!())
        .expect("erro ao iniciar o aplicativo");
}
