use crate::index::Index;

/// Decisão da comparação em três vias (base = último estado sincronizado neste PC).
#[derive(Debug, PartialEq, Eq)]
pub enum Action {
    Nothing,
    RestoreFromCloud,
    UploadLocal,
    Conflict,
}

pub fn decide(base: Option<&str>, local: &Index, cloud: Option<&str>) -> Action {
    let local_d = local.digest();
    // Sem base (PC novo), um save local vazio conta como "não mudou".
    let local_changed = match base {
        None => !local.files.is_empty(),
        Some(b) => b != local_d,
    };
    let cloud_changed = cloud != base;
    match (local_changed, cloud_changed) {
        (false, false) => Action::Nothing,
        (false, true) => Action::RestoreFromCloud,
        (true, false) => Action::UploadLocal,
        // Mesmo conteúdo nos dois lados: não há o que resolver.
        (true, true) if cloud == Some(local_d.as_str()) => Action::Nothing,
        (true, true) => Action::Conflict,
    }
}
