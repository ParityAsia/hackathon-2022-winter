use anyhow::{Error, Result};
use url::Url;

use move_core_types::account_address::AccountAddress;
use move_core_types::language_storage::{ModuleId, StructTag};
use move_core_types::resolver::{ModuleResolver, ResourceResolver};

#[cfg(feature = "dfinance")]
mod dnode;

mod pont;
use crate::pont::PontNet;

pub type Block = String;

pub fn make_net<T>(uri: T) -> Result<Box<dyn Net>>
where
    T: Into<Url>,
{
    let uri = uri.into();

    Ok(Box::new(PontNet {
        api: uri.to_string(),
    }))
}

#[derive(Debug)]
pub struct BytesForBlock(pub Vec<u8>, pub Block);

pub trait Net {
    fn get_module(
        &self,
        module_id: &ModuleId,
        height: &Option<Block>,
    ) -> Result<Option<BytesForBlock>>;

    fn get_resource(
        &self,
        address: &AccountAddress,
        tag: &StructTag,
        height: &Option<Block>,
    ) -> Result<Option<BytesForBlock>>;
}

pub struct NetView {
    net: Box<dyn Net>,
    block: Option<Block>,
}

impl NetView {
    pub fn new(net: Box<dyn Net>, block: Option<Block>) -> NetView {
        NetView { net, block }
    }

    pub fn set_block(&mut self, block: Option<Block>) {
        self.block = block;
    }
}

impl ModuleResolver for NetView {
    type Error = anyhow::Error;

    fn get_module(&self, module_id: &ModuleId) -> anyhow::Result<Option<Vec<u8>>> {
        self.net
            .get_module(module_id, &self.block)
            .map(|bytes| bytes.map(|bytes| bytes.0))
    }
}

impl ResourceResolver for NetView {
    type Error = Error;

    fn get_resource(&self, address: &AccountAddress, tag: &StructTag) -> Result<Option<Vec<u8>>> {
        self.net
            .get_resource(address, tag, &self.block)
            .map(|bytes| bytes.map(|bytes| bytes.0))
    }
}
