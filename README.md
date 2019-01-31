# rdrr

Regionally Distributed Runoff Recharge model is a large-scale high-resolution grid-based numerical model optimized for implementation on parallel computer architectures. No process of the model is in any way novel, rather a suite of existing model structures have been combined into one platform chosen specifically for their ease of implementation and computational scalability.

The model's primary intention is to interpolate watershed moisture conditions for the purposes of estimating regional groundwater recharge, given all available data. While the model can project stream flow discharge and overland runoff at any point in space, the model is not optimized, nor intended to serve as your typical rainfall runoff model. Outputs from this model will most often be used to constrain regional groundwater models withing the [Oak Ridges Moraine Groundwater Program (ORMGP)](https://oakridgeswater.ca) jurisdiction in southern Ontario, Canada.

This model is currently in beta mode, and its use for practical application should proceed with caution. Users should be aware that results posted online are subject to change. Ultimately, the intent of this model is to produced ranges of long-term (monthly and greater) water budgeting as a hydrological reference for the partners of the ORMGP.

### Model components:

- regional climate fields (model inputs: precipitation, temperature, etc.)
- explicit soil moisture accounting scheme
- cold content energy balance snowpack model
- distribution function-based shallow groundwater model
- local inertial approximation of the shallow water equation for lateral movement of water

*more details to come..*