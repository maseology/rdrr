


# Prepare model input files
> `prep/gobBuilder2/main.go`

1. read instruction file `M:/Peel/RDRR-PWRMM21/PWRMM21.rdrr`

1. save to go binaries (*.gob) for efficiency
    - Set main model file directory path (`gobDir`)
    - Load grid definition (`gdefFP`)


## Load model structure (`STRC`)
loads Hydrologically corrected Digital Elevation Model (HDEM) includes grid cell topography (upslope cells, outlet cell, etc.) (`hdemFP`)


### ADD **URBAN DIVERSION**


## Load sub-basins and routing topology (`RTR`)
Builds:
1. cross-referencing subbasin to/from grid cell IDs; 
1. downslope subbasin topography

(`swsFP`)

## Build land use, surficial geology and gw zone mapping (`MAPR`)
> luFPprfx, sgFP, gwzFP

Collects:
1. surface land use properties
    1. land use type (`surfaceid`)
    1. canopy type (`canopyid`)
    1. fraction ([0,1]) of impervious cover (`perimp`)
    1. fraction ([0,1]) of canopy cover (`percov`)

1. surficial/shallow geology

    material properties as 8 types (based on OGS characterization): 
    - [Low, Low-med, Medium, High-Med, High (permeability), Alluvium, Organic, Variable]

1. Ground water reservoirs

    Areas of distinct interconnected shallow groundwater systems, indexed (could be related to physiography). This is further broken down to:

    1. cell and stream cross-referencing
    1. TOPMODEL unit contributing areas $(a_i)$




These data are translated to a set of distributed (cell-based) model parameters:
1. Saturated conductivity of shallow soil zone $(K_\text{sat})$ 
1. Fraction imperviousness $(f_\text{imp})$
1. Land use
    - Depression storage $(h_\text{dep})$
    - Interception storage $(h_\text{can})$
    - Soil extinction depth $(z_\text{ext})$
    - Soil porosity $(\phi)$
    - Soil Field Capacity $(\theta_\text{fc})$


Cross-referencing from cells to land use, surficial geology and groundwater reservoir are made.



## Load sub-watershed forcings (`FORC`)
> midFP, ncfp

1. Builds cross-reference: met id to cell id
1. Reads timeseries arrays of meteorological forcings:
    1. atmospheric yield $(Y_a)$
    1. atmospheric demand) $(E_a)$



# Output

The rdrr prep creates 4 input files in the root directory:
- .domain.STRC.gob
- .domain.RTR.gob
- .domain.MAPR.gob
- .domain.FORC.gob

these are written as Go(lang) binary files.





# Code Structure (check)

The model (i.e., structural data, parameters, etc.) is structured as follows:

### Model domain
This is the overarching placeholder for all necessary data to run a model. The idea here is to collect data from a greater region that can be parsed to smaller areas where the numerical model is to be applied, say to some given basin outlet cell ID. This allows for a single greater set of input data to be consolidated into fewer computer files

### Subdomain
When the user wishes to run the model, there may be interest to specific areas and not the entire domain. For instance, user has only to specify the cell ID from which a catchment area drains to; or a grid definition with pre-defined active cells can be inputted.

### Sample
While the Domain and Subdomain carry mostly structural (i.e., unchanging) data, the sample is where a Subdomain is parameterized. Samples are handy when one attempts to optimize or perform a Monte Carlo analysis of parameter space using the same Subdomain.

### Subsample
For larger catchments, it may be advantageous to sub-divide the Subdomain for a particular parameter sample, say by subwatersheds of a predefined catchment area. This can be leveraged when scaling the model across multi-processor computer architectures. Subsample also holds sample results once the model has been evaluated.