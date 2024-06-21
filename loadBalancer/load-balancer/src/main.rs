use std::sync::mpsc;
use std::sync::mpsc::{Sender, Receiver};
use std::{thread, time};

fn main() {
    println!("-------- Starting Load Balancer --------");

    let mut balancer = LoadBalancer::new();

    balancer.add_balance_target(&NumberedConsoleWriter{ number: 1 });
    balancer.add_balance_target(&NumberedConsoleWriter{ number: 2 });
    balancer.add_balance_target(&NumberedConsoleWriter{ number: 3 });
    balancer.add_message("test 1");
    balancer.add_message("test 2");
    balancer.add_message("test 3");
    balancer.add_message("test 4");
    balancer.add_message("test 5");
    balancer.add_message("test 6");
    balancer.add_message("test 7");
    balancer.add_message("test 8");
    balancer.start();
    thread::sleep(time::Duration::from_millis(5000));
    balancer.stop();
}

trait BalanceTarget {
    fn write_message(&self, message: &str) -> bool;
}

pub struct NumberedConsoleWriter {
    pub number: i16,
}

impl BalanceTarget for NumberedConsoleWriter {
    fn write_message(&self, message: &str) -> bool {
        println!("{0} - {1}", self.number, message);
        return true;
    }
}

pub struct LoadBalancer<'bal> {
    balance_targets: Vec<&'bal dyn BalanceTarget>,
    last_target: usize,
    sender: Sender<&'bal str>,
    receiver: Receiver<&'bal str>,
    thread: Option<std::thread::JoinHandle<()>>,
    running: bool,
}

impl<'bal> LoadBalancer<'bal> {
    fn new() -> Self {
        let (tx, rx): (Sender<&str>, Receiver<&str>) =  mpsc::channel();
        return LoadBalancer {
            balance_targets: Vec::new(),
            last_target: 0,
            sender: tx,
            receiver: rx,
            thread: None,
            running: false,
        };
    }

    fn start(&mut self) {
        if !self.running {
            self.thread = Some(thread::spawn(move || {
                while self.running {
                    self.balance();
                }
            }));
        }
        self.running = true;
    }

    fn stop(&mut self) {
        self.thread.take().unwrap().join().unwrap();
        self.running = false;
    }

    fn add_balance_target(&mut self, target: &'bal (dyn BalanceTarget + 'bal)) {
        self.balance_targets.push(target);
    }

    fn add_message(&mut self, message: &'bal str) {
        self.sender.send(message).unwrap();
    }

    fn balance(&mut self) {
        let cur_target = (self.last_target + 1) % self.balance_targets.len();
        let target = self.balance_targets[cur_target];
        self.last_target = cur_target;
        target.write_message(self.receiver.recv().unwrap());
    }
}
