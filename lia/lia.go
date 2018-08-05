package lia

// LIA local inertial approximation to the 2D SWEs
// ref: de Almeida, G.A.M., P. Bates, 2013. Applicability of the local intertial approximation of the shallow water equations to flood modeling. Water Resources Research 49: 4833-4844.
// see also: de Almeida Bates Freer Souvignet 2012 Improving the stability of a simple formulation of the shallow water equations for 2-D flood modeling
//           Sampson etal 2012 An automated routing methodology to enable direct rainfall in high resolution shallow water models
// similar in theory to LISFLOOD-FP
type LIA struct {
 f map[int]face
 n map[int]node
 s map[int]state
 r map[int]float64 // vertical influx [m/s]
 fxr map[int][]int
 bf map[int]bool
 tacum, dt, dx, alpha, theta, tresid float64
}

// New constructor
func (m *LIA) New() {
	m.alpha = 0.7
	m.theta = 0.7
	m.tresid = 0.00001
}

 type state struct {

 }
 type node struct {

 }
type face struct {

}

        // Public ReadOnly Property GridFace As Grid.Face
        //     Get
        //         Return _gf
        //     End Get
        // End Property
        // Public Property alpha As Double
        //     Get
        //         Return _alpha
        //     End Get
        //     Set(value As Double)
        //         _alpha = value
        //     End Set
        // End Property
        // Public Property theta As Double
        //     Get
        //         Return _theta
        //     End Get
        //     Set(value As Double)
        //         _theta = value
        //         For Each f In _f.Values
        //             f.theta = _theta
        //         Next
        //     End Set
        // End Property
        // Public Property InFlux() As Dictionary(Of Integer, Double)
        //     Get
        //         Return _r
        //     End Get
        //     Set(ByVal value As Dictionary(Of Integer, Double))
        //         _r = value
        //     End Set
        // End Property

        Sub New(DEM As Grid.Real)
            Dim dicz As New Dictionary(Of Integer, Double), dicn As New Dictionary(Of Integer, Double), dich As New Dictionary(Of Integer, Double)
            For cid = 0 To DEM.GridDefinition.NumCells - 1
                dicz.Add(cid, DEM.Value(-9999, cid))
                dicn.Add(cid, 0.05)
                dich.Add(cid, dicz(cid))
            Next
            Me.Build(DEM.GridDefinition, dicz, dich, dicn)
        End Sub
        Sub New(DEM As Grid.Real, Mannings_n As Double)
            Dim dicz As New Dictionary(Of Integer, Double), dicn As New Dictionary(Of Integer, Double), dich As New Dictionary(Of Integer, Double)
            For Each cid In DEM.GridDefinition.Actives(True)
                dicz.Add(cid, DEM.Value(-9999, cid))
                dicn.Add(cid, Mannings_n)
                'dich.Add(cid, 0.00001 + dicz(cid))
                dich.Add(cid, dicz(cid))
            Next
            Me.Build(DEM.GridDefinition, dicz, dich, dicn)
        End Sub
        Sub New(DEM As Grid.Real, Mannings_n As Dictionary(Of Integer, Double))
            Dim dicz As New Dictionary(Of Integer, Double), dich As New Dictionary(Of Integer, Double)
            For cid = 0 To DEM.GridDefinition.NumCells - 1
                dicz.Add(cid, DEM.Value(-9999, cid))
                dich.Add(cid, dicz(cid))
            Next
            Me.Build(DEM.GridDefinition, dicz, dich, Mannings_n)
        End Sub
        Sub New(DEM As Grid.Real, h0 As Dictionary(Of Integer, Double), Mannings_n As Dictionary(Of Integer, Double))
            Dim dicz As New Dictionary(Of Integer, Double)
            For cid = 0 To DEM.GridDefinition.NumCells - 1
                dicz.Add(cid, DEM.Value(-9999, cid))
            Next
            Me.Build(DEM.GridDefinition, dicz, h0, Mannings_n)
        End Sub
        Private Sub Build(GD As Grid.Definition, z As Dictionary(Of Integer, Double), h0 As Dictionary(Of Integer, Double), Mannings_n As Dictionary(Of Integer, Double))
            If Not GD.IsUniform Then Stop
            _dx = GD.CellWidth(0)
            _gf = New Grid.Face(GD)
            _f = New Dictionary(Of Integer, _face)
            _fxr = New Dictionary(Of Integer, Integer())
            _bf = New Dictionary(Of Integer, Boolean)
            _n = New Dictionary(Of Integer, _node)
            _s = New Dictionary(Of Integer, _state)
            For Each n In Mannings_n
                _n.Add(n.Key, New _node With {.Elevation = z(n.Key), .Head = h0(n.Key), .Mannings_n = n.Value, .FaceID = _gf.CellFace(n.Key)})
            Next
            For i = 0 To _gf.nFaces - 1
                Dim fc1 As New _face(_gf, i)
                If fc1.IsInactive Then Continue For
                _bf.Add(i, fc1.IsBoundary)
                _f.Add(i, fc1)
                If Not fc1.IsBoundary Then
                    _fxr.Add(i, fc1.IdColl)
                    _s.Add(i, New _state)
                End If
            Next
            For Each f In _f
                If _bf(f.Key) Then Continue For
                Dim n = f.Value.NodeIDs
                f.Value.Initialize(_n(n(0)), _n(n(1)), _theta, _dx)
            Next
        End Sub

        'Function SetGhostNodes(bFace As List(Of Integer)) As List(Of Integer)
        '    Dim lstC As New List(Of Integer)
        '    For Each f In bFace
        '        If Not _bf(f) Then Stop ' only applicable to boundary faces
        '        With _f(f)
        '            If .NodeIDs(0) = -1 And .NodeIDs(1) = -1 Then
        '                Stop ' error
        '            ElseIf .NodeIDs(0) = -1 Then
        '                lstC.Add(_n.Count)
        '                .NodeFromID = _n.Count
        '                _s.Add(f, New _state)
        '                Dim bfd = IIf(_gf.IsUpwardFace(f), 1, 0), nid = .NodeIDs(1)
        '                _fxr.Add(f, { .NodeIDs(0), nid, _n(nid).FaceID(bfd)})
        '                _n.Add(_n.Count, New _node With {.Head = _n(nid).Elevation, .Elevation = _n(nid).Elevation - 0.001, .Mannings_n = _n(nid).Mannings_n})
        '                .Initialize(_n(.NodeIDs(0)), _n(nid), _theta, _dx)
        '            ElseIf .NodeIDs(1) = -1 Then
        '                lstC.Add(_n.Count)
        '                .NodeToID = _n.Count
        '                _s.Add(f, New _state)
        '                Dim bfd = IIf(_gf.IsUpwardFace(f), 0, 2), nid = .NodeIDs(0)
        '                _fxr.Add(f, {nid, .NodeIDs(1), _n(nid).FaceID(bfd)})
        '                _n.Add(_n.Count, New _node With {.Head = _n(nid).Elevation, .Elevation = _n(nid).Elevation - 0.001, .Mannings_n = _n(nid).Mannings_n})
        '                .Initialize(_n(nid), _n(.NodeIDs(1)), _theta, _dx)
        '            Else
        '                Stop ' error
        '            End If
        '            _bf(f) = False
        '        End With
        '    Next
        '    Return lstC
        'End Function

        Function SetHeadBC(faces As List(Of Integer), Value As Double) As List(Of Integer)
            Dim dic1 As New Dictionary(Of Integer, Double)
            For Each f In faces
                dic1.Add(f, Value)
            Next
            Return Me.SetHeadBC(dic1)
        End Function
        Function SetHeadBC(fbc As Dictionary(Of Integer, Double)) As List(Of Integer)
            Dim lstC As New List(Of Integer)
            For Each f In fbc
                If Not _bf(f.Key) Then Stop ' only applicable to boundary faces
                With _f(f.Key)
                    If .NodeIDs(0) = -1 And .NodeIDs(1) = -1 Then
                        Stop ' error
                    ElseIf .NodeIDs(0) = -1 Then
                        lstC.Add(_n.Count)
                        .NodeFromID = _n.Count
                        _s.Add(f.Key, New _state)
                        Dim bfd = IIf(_gf.IsUpwardFace(f.Key), 1, 0), nid = .NodeIDs(1)
                        _fxr.Add(f.Key, { .NodeIDs(0), nid, _n(nid).FaceID(bfd)})
                        _n.Add(_n.Count, New _node With {.Head = f.Value, .Elevation = _n(nid).Elevation - 0.001, .Mannings_n = _n(nid).Mannings_n}) ' ghost node
                        .Initialize(_n(.NodeIDs(0)), _n(nid), _theta, _dx)
                    ElseIf .NodeIDs(1) = -1 Then
                        lstC.Add(_n.Count)
                        .NodeToID = _n.Count
                        _s.Add(f.Key, New _state)
                        Dim bfd = IIf(_gf.IsUpwardFace(f.Key), 0, 2), nid = .NodeIDs(0)
                        _fxr.Add(f.Key, {nid, .NodeIDs(1), _n(nid).FaceID(bfd)})
                        _n.Add(_n.Count, New _node With {.Head = f.Value, .Elevation = _n(nid).Elevation - 0.001, .Mannings_n = _n(nid).Mannings_n}) ' ghost node
                        .Initialize(_n(nid), _n(.NodeIDs(1)), _theta, _dx)
                    Else
                        Stop ' error
                    End If
                    _bf(f.Key) = False
                End With
            Next
            Return lstC
        End Function
        Function SetFluxBC(fbc As Dictionary(Of Integer, Double)) As List(Of Integer)
            Dim lstC As New List(Of Integer)
            For Each f In fbc
                If Not _bf(f.Key) Then Stop ' only applicable to boundary faces
                With _f(f.Key)
                    If .NodeIDs(0) = -1 And .NodeIDs(1) = -1 Then
                        Stop ' error
                    ElseIf .NodeIDs(0) = -1 Then
                        .Flux = f.Value
                    ElseIf .NodeIDs(1) = -1 Then
                        .Flux = -f.Value
                    Else
                        Stop ' error
                    End If
                End With
                lstC.Add(f.Key)
            Next
            Return lstC
        End Function
        Function SetFluxBC(fs As List(Of Integer), Value As Double) As List(Of Integer)
            Dim dic1 As New Dictionary(Of Integer, Double)
            For Each f In fs
                dic1.Add(f, Value)
            Next
            Return Me.SetFluxBC(dic1)
        End Function

        Sub SetFlux(fs As List(Of Integer), Value As Double)
            For Each f In fs
                If Not _bf(f) Then Stop ' only applicable to boundary faces
                With _f(f)
                    If .NodeIDs(0) = -1 And .NodeIDs(1) = -1 Then
                        Stop ' error
                    ElseIf .NodeIDs(0) = -1 Then
                        .Flux = Value
                    ElseIf .NodeIDs(1) = -1 Then
                        .Flux = -Value
                    Else
                        Stop ' error
                    End If
                End With
            Next
        End Sub

        Sub SetHeads(ns As List(Of Integer), Value As Double)
            For Each n In ns
                _n(n).Head = Value
            Next
        End Sub
        Sub SetHeads(nh As Dictionary(Of Integer, Double))
            For Each n In nh
                _n(n.Key).Head = n.Value
            Next
        End Sub
        Sub SetHeads(h As Double)
            For Each n In _n.Values
                n.Head = h
            Next
        End Sub

        Public Function Solve() As Dictionary(Of Integer, Double)
            ' steady-state
            Dim sf As New Dictionary(Of Integer, _face)
            For Each f In _f
                If _bf(f.Key) Then Continue For
                sf.Add(f.Key, f.Value)
            Next
100:        Me.SetCurrentState()
            _tacum += _dt
            'For Each f In sf
            '    f.Value.UpdateFlux(_s(f.Key), _dt)
            'Next
            Parallel.ForEach(sf, Sub(f) f.Value.UpdateFlux(_s(f.Key), _dt))
            Dim r = Me.UpdateHeads
            Console.WriteLine("{0:0.00000}  {1:0.0000}", _tacum, r)
            If Math.Abs(r) > _tresid Then GoTo 100
            Dim dicOut As New Dictionary(Of Integer, Double)
            For Each n In _n
                If n.Key >= _gf.nCells Then Exit For ' ghost node boundary condition
                dicOut.Add(n.Key, n.Value.Head)
            Next
            Return dicOut
        End Function
        Public Function Solve(TimeStepSec As Double) As Dictionary(Of Integer, Double)
            _tacum = 0.0
            _dt = TimeStepSec
            Dim sf As New Dictionary(Of Integer, _face)
            For Each f In _f
                If _bf(f.Key) Then Continue For
                sf.Add(f.Key, f.Value)
            Next
            Do
                Me.SetCurrentState()
                _tacum += _dt
                If _tacum > TimeStepSec Then
                    _dt -= _tacum - TimeStepSec
                    _tacum = TimeStepSec
                End If
                'For Each f In sf
                '    f.Value.UpdateFlux(_s(f.Key), _dt)
                'Next
                Parallel.ForEach(sf, Sub(f) f.Value.UpdateFlux(_s(f.Key), _dt))
                Me.pUpdateHeads()
                Console.Write(".")
            Loop Until _tacum = TimeStepSec

            Dim dicOut As New Dictionary(Of Integer, Double)
            For Each n In _n
                If n.Key >= _gf.nCells Then Exit For ' ghost node boundary condition
                dicOut.Add(n.Key, n.Value.Head)
            Next
            Return dicOut
        End Function
        Public Function Velocities() As Dictionary(Of Integer, Double)
            Dim dicOut As New Dictionary(Of Integer, Double)
            For Each n In _n
                If n.Key >= _gf.nCells Then Exit For ' ghost node boundary condition
                With n.Value
                    If .Depth > 0.002 * _dx Then dicOut.Add(n.Key, Math.Sqrt(((_f(.FaceID(2)).Flux + _f(.FaceID(0)).Flux) / 2.0) ^ 2.0 + ((_f(.FaceID(3)).Flux + _f(.FaceID(1)).Flux) / 2.0) ^ 2.0) / .Depth) Else dicOut.Add(n.Key, 0.0)
                End With
            Next
            Return dicOut
        End Function

        Private Sub SetCurrentState()
            Dim dmax = Double.MinValue
            For Each n In _n.Values
                If n.Depth > dmax Then dmax = n.Depth
            Next
            If dmax > 0.0 Then _dt = _alpha * _dx / Math.Sqrt(9.80665 * dmax) ' eq.12
            'For Each s In _s
            '    With s.Value
            '        .N0h = _n(_fxr(s.Key)(0)).Head
            '        .N1h = _n(_fxr(s.Key)(1)).Head
            '        .bFlux = _f(_fxr(s.Key)(2)).Flux
            '        If _fxr(s.Key).Count = 3 Then ' ghost node boundary condition
            '            '.fFlux = _f(_fxr(s.Key)(2)).Flux
            '            .AvgOrthoFlux = 0.0
            '        Else
            '            .fFlux = _f(_fxr(s.Key)(3)).Flux
            '            Dim qorth As Double = 0.0
            '            For i = 4 To 7
            '                qorth += _f(_fxr(s.Key)(i)).Flux
            '            Next
            '            .AvgOrthoFlux = qorth / 4.0 ' eq. 9/10 average orthogonal flux
            '        End If
            '    End With
            'Next
            Parallel.ForEach(_s, Sub(s)
                                     With s.Value
                                         .N0h = _n(_fxr(s.Key)(0)).Head
                                         .N1h = _n(_fxr(s.Key)(1)).Head
                                         .bFlux = _f(_fxr(s.Key)(2)).Flux
                                         If _fxr(s.Key).Count = 3 Then ' ghost node boundary condition
                                             '.fFlux = _f(_fxr(s.Key)(2)).Flux
                                             .AvgOrthoFlux = 0.0
                                         Else
                                             .fFlux = _f(_fxr(s.Key)(3)).Flux
                                             Dim qorth As Double = 0.0
                                             For i = 4 To 7
                                                 qorth += _f(_fxr(s.Key)(i)).Flux
                                             Next
                                             .AvgOrthoFlux = qorth / 4.0 ' eq. 9/10 average orthogonal flux
                                         End If
                                     End With
                                 End Sub)
        End Sub
        Private Function UpdateHeads() As Double
            Dim resid As Double = 0.0, aresid As Double = 0.0, d1 = _dt / _dx '^ 2.0 ' error in equation 11, see eq. 20 in de Almeda etal 2012
            For Each n In _n
                If n.Key >= _gf.nCells Then Exit For ' ghost node boundary condition
                With n.Value
                    Dim dh = d1 * (_f(.FaceID(2)).Flux - _f(.FaceID(0)).Flux + _f(.FaceID(3)).Flux - _f(.FaceID(1)).Flux) ' eq 11
                    If Not IsNothing(_r) AndAlso _r.ContainsKey(n.Key) Then dh += _r(n.Key)
                    Dim adh = Math.Abs(dh)
                    If adh > aresid Then
                        aresid = adh
                        resid = dh
                    End If
                    .Head += dh
                End With
            Next
            Return resid
        End Function
        Private Sub pUpdateHeads()
            Dim d1 = _dt / _dx '^ 2.0 ' error in equation 11, see eq. 20 in de Almeda etal 2012
            Parallel.ForEach(_n, Sub(n)
                                     If n.Key < _gf.nCells Then ' ghost node boundary condition
                                         With n.Value
                                             .Head += d1 * (_f(.FaceID(2)).Flux - _f(.FaceID(0)).Flux + _f(.FaceID(3)).Flux - _f(.FaceID(1)).Flux) ' eq 11
                                         End With
                                     End If
                                 End Sub)
            If Not IsNothing(_r) Then
                Parallel.ForEach(_r, Sub(r)
                                         With _n(r.Key)
                                             .Head += r.Value * _dt ' m
                                         End With
                                     End Sub)
            End If
        End Sub

        Private Class _node
            Private _z As Double, _h As Double, _n As Double, _f As Integer()

            Public Property Elevation() As Double
                Get
                    Return _z
                End Get
                Set(ByVal value As Double)
                    _z = value
                End Set
            End Property
            Public Property Head() As Double
                Get
                    Return _h
                End Get
                Set(ByVal value As Double)
                    _h = value
                    If value < _z Then _h = _z
                End Set
            End Property
            Public Property Mannings_n() As Double
                Get
                    Return _n
                End Get
                Set(ByVal value As Double)
                    _n = value
                End Set
            End Property
            Public ReadOnly Property Depth As Double
                Get
                    Return _h - _z
                End Get
            End Property
            Public Property FaceID As Integer()
                Get
                    Return _f
                End Get
                Set(ByVal value As Integer())
                    _f = value
                End Set
            End Property
        End Class
        Private Class _face
            Private _nfrom As Integer, _nto As Integer, _ffw As Integer = -1, _fbw As Integer = -1, _forth As Integer() ' node and face identifiers
            Private _t As Double, _dx As Double, _n2 As Double, _zx As Double ' parameters
            Private _q As Double ' varables

            Public Property theta As Double
                Get
                    Return _t
                End Get
                Set(value As Double)
                    _t = value
                End Set
            End Property
            Public Property Flux As Double
                Get
                    Return _q
                End Get
                Set(value As Double)
                    _q = value
                End Set
            End Property
            Public ReadOnly Property NodeIDs As Integer()
                Get
                    Return {_nfrom, _nto}
                End Get
            End Property
            Public Property NodeFromID As Integer
                Get
                    Return _nfrom
                End Get
                Set(value As Integer)
                    _nfrom = value
                End Set
            End Property
            Public Property NodeToID As Integer
                Get
                    Return _nto
                End Get
                Set(value As Integer)
                    _nto = value
                End Set
            End Property
            Public ReadOnly Property IsBoundary As Boolean
                Get
                    Return IsNothing(_forth)
                End Get
            End Property
            Public ReadOnly Property IsInactive As Boolean
                Get
                    Return _nfrom = -1 AndAlso _nto = -1
                End Get
            End Property
            Public ReadOnly Property IdColl As Integer()
                Get
                    Dim in1(7) As Integer
                    in1(0) = _nfrom
                    in1(1) = _nto
                    in1(2) = _fbw
                    in1(3) = _ffw
                    For i = 0 To 3
                        in1(4 + i) = _forth(i)
                    Next
                    Return in1
                End Get
            End Property

            Sub New()
            End Sub
            Sub New(GF As Grid.Face, fid As Integer)
                With GF
                    _nfrom = .FaceCell(fid)(0)
                    _nto = .FaceCell(fid)(1)
                    If _nfrom = -1 Or _nto = -1 Then
                        _q = 0 ' (default) no flow boundary
                    Else
                        ReDim _forth(3) ' orthogonal faces
                        If .IsUpwardFace(fid) Then ' upward meaning direction normal to face
                            _ffw = .CellFace(_nto)(1)
                            _fbw = .CellFace(_nfrom)(3)
                            _forth(0) = .CellFace(_nfrom)(2)
                            _forth(1) = .CellFace(_nfrom)(0)
                            _forth(2) = .CellFace(_nto)(2)
                            _forth(3) = .CellFace(_nto)(0)
                        Else
                            _ffw = .CellFace(_nto)(0)
                            _fbw = .CellFace(_nfrom)(2)
                            _forth(0) = .CellFace(_nfrom)(3)
                            _forth(1) = .CellFace(_nfrom)(1)
                            _forth(2) = .CellFace(_nto)(3)
                            _forth(3) = .CellFace(_nto)(1)
                        End If
                    End If
                End With
            End Sub

            Sub Initialize(Node0 As _node, Node1 As _node, Theta As Double, cellsize As Double)
                _t = Theta
                _dx = cellsize
                _zx = Math.Max(Node0.Elevation, Node1.Elevation)
                _n2 = ((Node0.Mannings_n + Node1.Mannings_n) / 2.0) ^ 2.0
            End Sub

            Public Sub UpdateFlux(s As _state, dt As Double)
                With s
                    Dim hf = Math.Max(.N0h, .N1h) - _zx
                    If hf <= 0.000001 Then
                        _q = 0.0
                    Else
                        Dim qmag = Math.Sqrt(_q ^ 2.0 + .AvgOrthoFlux ^ 2.0) ' eq. 8
                        'Dim qmag = Math.Abs(_q) ' de Almeda etal 2012
                        _q = _t * _q + 0.5 * (1.0 - _t) * (.fFlux + .bFlux) - 9.80665 * hf * dt * (.N1h - .N0h) / _dx ' eq. 7 numer
                        _q /= 1 + 9.80665 * dt * _n2 * qmag / hf ^ 2.33333 ' eq.7 denom
                    End If
                End With
            End Sub

        End Class

        Private Class _state
            Public Property N0h As Double
            Public Property N1h As Double
            Public Property fFlux As Double
            Public Property bFlux As Double
            Public Property AvgOrthoFlux As Double
        End Class