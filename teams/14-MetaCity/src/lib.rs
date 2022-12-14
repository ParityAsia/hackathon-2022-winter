#![cfg_attr(not(feature = "std"), no_std)]

//this is a proof of concept demo for a blockchain game called MetaCity.
//In this demo, users can choose different strategies to maximize their returns in the game.

//generally speaking, if an user want to pay a lot of time to play the game,
//She/He should better choose a mode that provide highest per unit time return but loweset valid duration time, otherwise the opposite.

use ink_lang as ink;

#[ink::contract]
mod game {

    #[ink(storage)]
    pub struct Game {
        //hit_point, just an attribute data for the user's player
        hp: u32,
        //attack_point, just an attribute data for the user's player
        atk: u32,
        //defend_point, just an attribute data for the user's player
        def: u32,
        //save the blocknumber since the last collect_energy function has run
        start_block: u32,
        //when set the mode to 1, get highest per unit time return but loweset valid duration time,
        //when set it to 2, get medium unit time return and medium valid duration time,
        //when set it to 3, get loweset per unit time return but highest valid duration time,
        mode: u32,
        //use for increase hp/atk/def, generated by function collect_energy
        energy: u32,
    }

    #[ink(event)]
    pub struct Event {
        //for saving storage in blockchain, we use integer instead of String to represent messages.
        error: u32,
        energy: u32,
    }

    impl Game {
        #[ink(constructor)]
        pub fn new(init_hp: u32, init_atk: u32, init_def: u32) -> Self {
            Self {
                hp: init_hp,
                atk: init_atk,
                def: init_def,
                start_block: 0,
                mode: 1,
                energy: 0,
            }
        }

        #[ink(message)]
        pub fn add_hp(&mut self, qty: u32) {
            if qty <= self.energy {
                self.hp += 10 * qty;
                self.energy -= qty;
            } else {
                self.env().emit_event(Event {
                    error: 1,
                    energy: self.energy,
                });
            }
        }

        #[ink(message)]
        pub fn add_atk(&mut self, qty: u32) {
            if qty <= self.energy {
                self.atk += qty;
                self.energy -= qty;
            } else {
                self.env().emit_event(Event {
                    error: 1,
                    energy: self.energy,
                });
            }
        }

        #[ink(message)]
        pub fn add_def(&mut self, qty: u32) {
            if qty <= self.energy {
                self.def += qty;
                self.energy -= qty;
            } else {
                self.env().emit_event(Event {
                    error: 1,
                    energy: self.energy,
                });
            }
        }

        #[ink(message)]
        pub fn get_hp(&self) -> u32 {
            self.hp
        }

        #[ink(message)]
        pub fn get_atk(&self) -> u32 {
            self.atk
        }

        #[ink(message)]
        pub fn get_def(&self) -> u32 {
            self.def
        }

        #[ink(message)]
        pub fn get_energy(&self) -> u32 {
            self.energy
        }

        #[ink(message)]
        pub fn get_block(&self) -> u32 {
            self.env().block_number()
        }

        #[ink(message)]
        pub fn get_startblock(&self) -> u32 {
            self.start_block
        }

        #[ink(message)]
        pub fn set_mode1(&mut self) {
            self.mode = 1
        }

        #[ink(message)]
        pub fn set_mode2(&mut self) {
            self.mode = 2
        }

        #[ink(message)]
        pub fn set_mode3(&mut self) {
            self.mode = 3
        }

        #[ink(message)]
        pub fn collect_energy(&mut self) {
            //set the duration blocks between now and the last time to call this function as 'duration'.
            let duration = self.env().block_number() - self.start_block;
            if self.mode == 1 {
                //in mode1, if the duration is less than 5, it get a highest per unit time return. But if it's no less than 5, it can only get 50 energy.
                //so in this mode1, users should prequently play game to maximize their returns
                if duration < 5 {
                    self.energy += 10 * duration
                } else {
                    self.energy += 50
                }
            } else if self.mode == 2 {
                if duration < 10 {
                    self.energy += 7 * duration
                } else {
                    self.energy += 70
                }
            } else if self.mode == 3 {
                //in mode3, if the duration is less than 20, it get a lowest per unit time return. But if it's no less than 20, it can only get 100 energy.
                //so for users who don't want to pay much time to play this game, mode3 is the best strategy for them.
                if duration < 20 {
                    self.energy += 5 * duration
                } else {
                    self.energy += 100
                }
            }
            self.start_block = self.env().block_number();
        }
    }
}
