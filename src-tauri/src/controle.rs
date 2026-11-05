//! Leitura do controle (Xbox, PlayStation etc.) no Rust, com `gilrs`.
//! Os botões viram as mesmas "ações" do teclado e chegam à interface pelo evento `controle-acao`.
//! Só valem quando a janela do app está visível e focada: um controle usado dentro de um jogo
//! nunca deve clicar em "Remover" no launcher que está escondido.

use gilrs::{Axis, Button, EventType, Gilrs};
use serde::Serialize;
use std::collections::HashMap;
use std::time::{Duration, Instant};
use tauri::{AppHandle, Emitter, Manager};

const PRIMEIRA_REPETICAO: Duration = Duration::from_millis(350);
const REPETICAO: Duration = Duration::from_millis(80);
const LIMIAR_ANALOGICO: f32 = 0.5;

#[derive(Serialize, Clone)]
struct AcaoDoControle {
    acao: &'static str,
    /// "xbox" ou "ps", para a interface mostrar os botões certos.
    dispositivo: &'static str,
}

pub fn acao_do_botao(b: Button) -> Option<&'static str> {
    Some(match b {
        Button::South => "confirm",
        Button::East => "back",
        Button::West => "filter",
        Button::North => "options",
        Button::LeftTrigger => "prevTab",
        Button::RightTrigger => "nextTab",
        Button::LeftTrigger2 => "pageUp",
        Button::RightTrigger2 => "pageDown",
        Button::Select => "search",
        Button::Start => "menu",
        Button::Mode | Button::RightThumb => "quick",
        Button::LeftThumb => "help",
        Button::DPadUp => "up",
        Button::DPadDown => "down",
        Button::DPadLeft => "left",
        Button::DPadRight => "right",
        _ => return None,
    })
}

fn repete(acao: &str) -> bool {
    matches!(acao, "up" | "down" | "left" | "right" | "pageUp" | "pageDown")
}

/// Direção que um eixo analógico indica (ou `None` se está perto do centro).
pub fn direcao_do_eixo(eixo: Axis, valor: f32) -> Option<&'static str> {
    match eixo {
        Axis::LeftStickX if valor > LIMIAR_ANALOGICO => Some("right"),
        Axis::LeftStickX if valor < -LIMIAR_ANALOGICO => Some("left"),
        // No gilrs, para cima é positivo.
        Axis::LeftStickY if valor > LIMIAR_ANALOGICO => Some("up"),
        Axis::LeftStickY if valor < -LIMIAR_ANALOGICO => Some("down"),
        _ => None,
    }
}

/// Controla quais ações estão seguradas e quando repetir. O tempo vem de fora, para testar.
#[derive(Default)]
pub struct Repetidor {
    segurados: HashMap<&'static str, Instant>,
}

impl Repetidor {
    /// Registra um aperto. Devolve `true` se a ação deve ser emitida agora.
    pub fn apertar(&mut self, acao: &'static str, agora: Instant) -> bool {
        if self.segurados.contains_key(acao) {
            return false;
        }
        if repete(acao) {
            self.segurados.insert(acao, agora + PRIMEIRA_REPETICAO);
        } else {
            // Ações sem repetição ficam marcadas até soltar, para não disparar duas vezes.
            self.segurados.insert(acao, agora + Duration::from_secs(3600));
        }
        true
    }

    pub fn soltar(&mut self, acao: &'static str) {
        self.segurados.remove(acao);
    }

    /// Ações que devem repetir agora.
    pub fn repeticoes(&mut self, agora: Instant) -> Vec<&'static str> {
        let mut saida = Vec::new();
        for (acao, proximo) in self.segurados.iter_mut() {
            if repete(acao) && agora >= *proximo {
                *proximo = agora + REPETICAO;
                saida.push(*acao);
            }
        }
        saida
    }
}

fn tipo_do_controle(nome: &str) -> &'static str {
    let n = nome.to_lowercase();
    if n.contains("dualsense") || n.contains("dualshock") || n.contains("playstation") || n.contains("wireless controller") || n.contains("054c") {
        "ps"
    } else {
        "xbox"
    }
}

fn janela_pronta(app: &AppHandle) -> bool {
    app.get_webview_window("main").is_some_and(|w| w.is_visible().unwrap_or(false) && w.is_focused().unwrap_or(false))
}

/// Inicia a leitura em segundo plano. Sem suporte a controle neste sistema, o app segue só com teclado e mouse.
pub fn iniciar(app: AppHandle) {
    std::thread::spawn(move || {
        let Ok(mut gilrs) = Gilrs::new() else { return };
        let mut rep = Repetidor::default();
        let mut direcao_analogica: HashMap<Axis, &'static str> = HashMap::new();
        let mut dispositivo = "xbox";
        loop {
            while let Some(ev) = gilrs.next_event() {
                dispositivo = tipo_do_controle(gilrs.gamepad(ev.id).name());
                let agora = Instant::now();
                match ev.event {
                    EventType::ButtonPressed(b, _) => {
                        if let Some(a) = acao_do_botao(b) {
                            if rep.apertar(a, agora) {
                                emitir(&app, a, dispositivo);
                            }
                        }
                    }
                    EventType::ButtonReleased(b, _) => {
                        if let Some(a) = acao_do_botao(b) {
                            rep.soltar(a);
                        }
                    }
                    EventType::AxisChanged(eixo, valor, _) => {
                        let nova = direcao_do_eixo(eixo, valor);
                        let antiga = direcao_analogica.get(&eixo).copied();
                        if nova != antiga {
                            if let Some(a) = antiga {
                                rep.soltar(a);
                            }
                            match nova {
                                Some(a) => {
                                    direcao_analogica.insert(eixo, a);
                                    if rep.apertar(a, agora) {
                                        emitir(&app, a, dispositivo);
                                    }
                                }
                                None => {
                                    direcao_analogica.remove(&eixo);
                                }
                            }
                        }
                    }
                    EventType::Disconnected => {
                        rep = Repetidor::default();
                        direcao_analogica.clear();
                    }
                    _ => {}
                }
            }
            for a in rep.repeticoes(Instant::now()) {
                emitir(&app, a, dispositivo);
            }
            std::thread::sleep(Duration::from_millis(8));
        }
    });
}

fn emitir(app: &AppHandle, acao: &'static str, dispositivo: &'static str) {
    if janela_pronta(app) {
        let _ = app.emit("controle-acao", AcaoDoControle { acao, dispositivo });
    }
}

#[cfg(test)]
mod testes {
    use super::*;

    #[test]
    fn botoes_viram_acoes() {
        assert_eq!(acao_do_botao(Button::South), Some("confirm"));
        assert_eq!(acao_do_botao(Button::East), Some("back"));
        assert_eq!(acao_do_botao(Button::DPadLeft), Some("left"));
        assert_eq!(acao_do_botao(Button::C), None);
    }

    #[test]
    fn analogico_usa_limiar_e_cima_e_positivo() {
        assert_eq!(direcao_do_eixo(Axis::LeftStickX, 0.9), Some("right"));
        assert_eq!(direcao_do_eixo(Axis::LeftStickX, -0.9), Some("left"));
        assert_eq!(direcao_do_eixo(Axis::LeftStickY, 0.9), Some("up"));
        assert_eq!(direcao_do_eixo(Axis::LeftStickY, -0.9), Some("down"));
        assert_eq!(direcao_do_eixo(Axis::LeftStickX, 0.3), None, "zona morta");
        assert_eq!(direcao_do_eixo(Axis::RightStickX, 0.9), None);
    }

    #[test]
    fn direcional_repete_depois_de_um_atraso_e_confirmar_nao() {
        let t0 = Instant::now();
        let mut r = Repetidor::default();
        assert!(r.apertar("right", t0));
        assert!(!r.apertar("right", t0), "segurar não emite de novo");
        assert!(r.repeticoes(t0 + Duration::from_millis(100)).is_empty());
        assert_eq!(r.repeticoes(t0 + Duration::from_millis(400)), vec!["right"]);
        assert!(r.repeticoes(t0 + Duration::from_millis(450)).is_empty(), "espera o intervalo");
        assert_eq!(r.repeticoes(t0 + Duration::from_millis(500)), vec!["right"]);
        r.soltar("right");
        assert!(r.repeticoes(t0 + Duration::from_secs(5)).is_empty());

        assert!(r.apertar("confirm", t0));
        assert!(r.repeticoes(t0 + Duration::from_secs(5)).is_empty(), "confirmar não repete");
        r.soltar("confirm");
        assert!(r.apertar("confirm", t0), "pode apertar de novo depois de soltar");
    }

    #[test]
    fn reconhece_controle_playstation() {
        assert_eq!(tipo_do_controle("Sony DualSense Wireless Controller"), "ps");
        assert_eq!(tipo_do_controle("Xbox Series X Controller"), "xbox");
    }
}
